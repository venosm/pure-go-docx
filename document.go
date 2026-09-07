package godocx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path"
	"sort"
	"strings"

	"github.com/venosm/pure-go-docx/internal/body"
	"github.com/venosm/pure-go-docx/internal/numbering"
	"github.com/venosm/pure-go-docx/internal/opc"
)

// Document is the parsed representation of a DOCX file.
type Document struct {
	Body      []Block
	Headers   map[string][]Block
	Footers   map[string][]Block
	Footnotes map[string][]Block
	Endnotes  map[string][]Block

	pkg       *opc.Package
	mainPart  string
	partRels  map[string]map[string]opc.Relationship
	imageRefs map[string]ImageRef
	numbering *numbering.Resolver
}

func openReader(r io.ReaderAt, size int64) (*Document, error) {
	pkg, err := opc.OpenReader(r, size)
	if err != nil {
		return nil, err
	}

	mainPart, err := pkg.MainDocumentPart()
	if err != nil {
		return nil, err
	}

	rels, err := pkg.Relationships(mainPart)
	if err != nil {
		return nil, err
	}

	resolver, err := parseNumbering(pkg, rels)
	if err != nil {
		return nil, err
	}

	data, err := pkg.ReadXMLPart(mainPart)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resolver.Reset()
	blocks, err := body.ParseDocument(ctx, bytes.NewReader(data), rels, resolver, mainPart)
	if err != nil {
		return nil, fmt.Errorf("parsing main document %q: %w", mainPart, err)
	}

	d := &Document{
		Body:      blocks,
		Headers:   make(map[string][]Block),
		Footers:   make(map[string][]Block),
		Footnotes: make(map[string][]Block),
		Endnotes:  make(map[string][]Block),
		pkg:       pkg,
		mainPart:  mainPart,
		partRels: map[string]map[string]opc.Relationship{
			mainPart: rels,
		},
		imageRefs: make(map[string]ImageRef),
		numbering: resolver,
	}
	d.collectImageRefs(blocks)
	if err := d.parseRelatedParts(ctx, rels); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Document) parseRelatedParts(ctx context.Context, rels map[string]opc.Relationship) error {
	for _, rel := range sortedRelationships(rels) {
		if rel.TargetMode == "External" {
			continue
		}
		switch rel.Type {
		case opc.HeaderRelationshipType:
			blocks, err := d.parseBlockPart(ctx, rel.Target, body.ParseHeader)
			if err != nil {
				return fmt.Errorf("parsing header %q: %w", rel.Target, err)
			}
			d.Headers[rel.ID] = blocks
		case opc.FooterRelationshipType:
			blocks, err := d.parseBlockPart(ctx, rel.Target, body.ParseFooter)
			if err != nil {
				return fmt.Errorf("parsing footer %q: %w", rel.Target, err)
			}
			d.Footers[rel.ID] = blocks
		case opc.FootnotesRelationshipType:
			notes, err := d.parseNotePart(ctx, rel.Target, body.ParseFootnotes)
			if err != nil {
				return fmt.Errorf("parsing footnotes %q: %w", rel.Target, err)
			}
			for id, blocks := range notes {
				d.Footnotes[id] = blocks
			}
		case opc.EndnotesRelationshipType:
			notes, err := d.parseNotePart(ctx, rel.Target, body.ParseEndnotes)
			if err != nil {
				return fmt.Errorf("parsing endnotes %q: %w", rel.Target, err)
			}
			for id, blocks := range notes {
				d.Endnotes[id] = blocks
			}
		}
	}
	return nil
}

type blockPartParser func(context.Context, io.Reader, map[string]opc.Relationship, *numbering.Resolver, string) ([]Block, error)
type notePartParser func(context.Context, io.Reader, map[string]opc.Relationship, *numbering.Resolver, string) (map[string][]Block, error)

func (d *Document) parseBlockPart(ctx context.Context, partName string, parse blockPartParser) ([]Block, error) {
	rels, err := d.pkg.Relationships(partName)
	if err != nil {
		return nil, err
	}
	d.partRels[partName] = rels

	data, err := d.pkg.ReadXMLPart(partName)
	if err != nil {
		return nil, err
	}

	d.numbering.Reset()
	blocks, err := parse(ctx, bytes.NewReader(data), rels, d.numbering, partName)
	if err != nil {
		return nil, err
	}
	d.collectImageRefs(blocks)
	return blocks, nil
}

func (d *Document) parseNotePart(ctx context.Context, partName string, parse notePartParser) (map[string][]Block, error) {
	rels, err := d.pkg.Relationships(partName)
	if err != nil {
		return nil, err
	}
	d.partRels[partName] = rels

	data, err := d.pkg.ReadXMLPart(partName)
	if err != nil {
		return nil, err
	}

	d.numbering.Reset()
	notes, err := parse(ctx, bytes.NewReader(data), rels, d.numbering, partName)
	if err != nil {
		return nil, err
	}
	for _, blocks := range notes {
		d.collectImageRefs(blocks)
	}
	return notes, nil
}

func parseNumbering(pkg *opc.Package, rels map[string]opc.Relationship) (*numbering.Resolver, error) {
	for _, rel := range rels {
		if rel.Type != opc.NumberingRelationshipType {
			continue
		}
		if rel.TargetMode == "External" {
			continue
		}
		data, err := pkg.ReadXMLPart(rel.Target)
		if err != nil {
			return nil, fmt.Errorf("reading numbering part %q: %w", rel.Target, err)
		}
		resolver, err := numbering.Parse(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("parsing numbering part %q: %w", rel.Target, err)
		}
		return resolver, nil
	}

	resolver, err := numbering.Parse(nil)
	if err != nil {
		return nil, fmt.Errorf("creating empty numbering resolver: %w", err)
	}
	return resolver, nil
}

// Image returns image bytes by relationship id. The image is read lazily.
//
// Body images can be loaded by their relationship id for compatibility.
// Images in related parts use "part/name.xml#rId" IDs.
func (d *Document) Image(id string) (Image, error) {
	partName, relID, explicitPart := splitImageID(id)
	if partName == "" {
		partName = d.mainPart
	}
	rel, ok := d.relationship(partName, relID)
	if !ok && !explicitPart {
		if ref, unique := d.uniqueImageRef(relID); unique {
			return d.imageFromRef(ref)
		}
	}
	if !ok {
		return Image{}, fmt.Errorf("image relationship %q not found in part %q", relID, partName)
	}
	if rel.TargetMode == "External" {
		return Image{}, fmt.Errorf("image relationship %q is external", relID)
	}
	if rel.Type != "" && rel.Type != opc.ImageRelationshipType {
		return Image{}, fmt.Errorf("relationship %q is not an image: %s", relID, rel.Type)
	}

	r, err := d.pkg.OpenPart(rel.Target)
	if err != nil {
		return Image{}, err
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return Image{}, fmt.Errorf("reading image %q: %w", rel.Target, err)
	}

	image := Image{
		ID:          imageKey(partName, relID),
		PartName:    partName,
		RelID:       relID,
		ContentType: d.imageContentType(rel.Target),
		Filename:    rel.Target,
		Bytes:       data,
	}
	if ref, ok := d.imageRefs[image.ID]; ok {
		image.AltText = ref.AltText
	}
	return image, nil
}

// AllImages returns every image referenced from the body and related parts.
func (d *Document) AllImages() ([]Image, error) {
	ids := sortedStringKeys(d.imageRefs)
	images := make([]Image, 0, len(ids))
	for _, id := range ids {
		image, err := d.imageFromRef(d.imageRefs[id])
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

func (d *Document) imageFromRef(ref ImageRef) (Image, error) {
	partName := ref.PartName
	if partName == "" {
		partName = d.mainPart
	}
	image, err := d.Image(imageKey(partName, ref.RelID))
	if err != nil {
		return Image{}, err
	}
	if image.AltText == "" {
		image.AltText = ref.AltText
	}
	return image, nil
}

func (d *Document) collectImageRefs(blocks []Block) {
	for _, block := range blocks {
		switch b := block.(type) {
		case *Paragraph:
			for _, run := range b.Runs {
				if run.Image != nil {
					key := imageRefKey(*run.Image)
					if _, ok := d.imageRefs[key]; !ok {
						ref := *run.Image
						ref.ID = key
						d.imageRefs[key] = ref
					}
				}
			}
		case *Table:
			for _, row := range b.Grid {
				for _, cell := range row {
					d.collectImageRefs(cell.Blocks)
				}
			}
		}
	}
}

func (d *Document) relationship(partName, relID string) (opc.Relationship, bool) {
	if partName == "" {
		partName = d.mainPart
	}
	if rels, ok := d.partRels[partName]; ok {
		rel, ok := rels[relID]
		return rel, ok
	}
	return opc.Relationship{}, false
}

func (d *Document) uniqueImageRef(relID string) (ImageRef, bool) {
	var found ImageRef
	count := 0
	for _, ref := range d.imageRefs {
		if ref.RelID != relID {
			continue
		}
		found = ref
		count++
	}
	return found, count == 1
}

func (d *Document) imageContentType(filename string) string {
	if contentType := d.pkg.ContentType(filename); contentType != "" {
		return contentType
	}
	if contentType := mime.TypeByExtension(path.Ext(filename)); contentType != "" {
		return strings.Split(contentType, ";")[0]
	}
	switch strings.ToLower(path.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

func sortedRelationships(rels map[string]opc.Relationship) []opc.Relationship {
	ids := make([]string, 0, len(rels))
	for id := range rels {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	relationships := make([]opc.Relationship, 0, len(ids))
	for _, id := range ids {
		relationships = append(relationships, rels[id])
	}
	return relationships
}

func sortedStringKeys(values map[string]ImageRef) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func imageRefKey(ref ImageRef) string {
	if ref.ID != "" {
		return ref.ID
	}
	return imageKey(ref.PartName, ref.RelID)
}

func imageKey(partName, relID string) string {
	if partName == "" {
		return relID
	}
	return partName + "#" + relID
}

func splitImageID(id string) (partName, relID string, explicitPart bool) {
	index := strings.LastIndex(id, "#")
	if index < 0 {
		return "", id, false
	}
	return id[:index], id[index+1:], true
}
