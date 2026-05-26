package godocx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path"
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

	pkg       *opc.Package
	bodyRels  map[string]opc.Relationship
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

	blocks, err := body.Parse(context.Background(), bytes.NewReader(data), rels, resolver)
	if err != nil {
		return nil, fmt.Errorf("parsing main document %q: %w", mainPart, err)
	}

	// TODO(milestone-3): call resolver.Reset() before walking each header/footer/footnote part.
	d := &Document{
		Body:      blocks,
		Headers:   make(map[string][]Block),
		Footers:   make(map[string][]Block),
		Footnotes: make(map[string][]Block),
		pkg:       pkg,
		bodyRels:  rels,
		imageRefs: make(map[string]ImageRef),
		numbering: resolver,
	}
	d.collectImageRefs(blocks)
	return d, nil
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
func (d *Document) Image(relID string) (Image, error) {
	rel, ok := d.bodyRels[relID]
	if !ok {
		return Image{}, fmt.Errorf("image relationship %q not found", relID)
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
		RelID:       relID,
		ContentType: d.imageContentType(rel.Target),
		Filename:    rel.Target,
		Bytes:       data,
	}
	if ref, ok := d.imageRefs[relID]; ok {
		image.AltText = ref.AltText
	}
	return image, nil
}

// AllImages returns every image referenced from the body, headers, and footers.
func (d *Document) AllImages() ([]Image, error) {
	ids := imageIDs(d.Body)
	images := make([]Image, 0, len(ids))
	for _, id := range ids {
		image, err := d.Image(id)
		if err != nil {
			return nil, err
		}
		images = append(images, image)
	}
	return images, nil
}

func (d *Document) collectImageRefs(blocks []Block) {
	for _, block := range blocks {
		switch b := block.(type) {
		case *Paragraph:
			for _, run := range b.Runs {
				if run.Image != nil {
					if _, ok := d.imageRefs[run.Image.RelID]; !ok {
						d.imageRefs[run.Image.RelID] = *run.Image
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

func imageIDs(blocks []Block) []string {
	seen := make(map[string]bool)
	var ids []string
	var walk func([]Block)
	walk = func(blocks []Block) {
		for _, block := range blocks {
			switch b := block.(type) {
			case *Paragraph:
				for _, run := range b.Runs {
					if run.Image == nil || seen[run.Image.RelID] {
						continue
					}
					seen[run.Image.RelID] = true
					ids = append(ids, run.Image.RelID)
				}
			case *Table:
				for _, row := range b.Grid {
					for _, cell := range row {
						walk(cell.Blocks)
					}
				}
			}
		}
	}
	walk(blocks)
	return ids
}
