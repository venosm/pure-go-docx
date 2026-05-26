package opc

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
)

const (
	maxParts             = 10_000
	maxUncompressedBytes = 500 << 20
	maxXMLPartBytes      = 64 << 20

	contentTypesPart = "[Content_Types].xml"
	mainDocumentType = "wordprocessingml.document.main+xml"
)

// Package is an OPC package backed by a DOCX zip archive.
type Package struct {
	reader       *zip.Reader
	files        map[string]*zip.File
	defaultTypes map[string]string
	overrides    map[string]string
}

// OpenReader opens an OPC package from a random-access reader.
func OpenReader(r io.ReaderAt, size int64) (*Package, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("opening zip package: %w", err)
	}
	if len(zr.File) > maxParts {
		return nil, fmt.Errorf("too many zip parts: %d > %d", len(zr.File), maxParts)
	}

	var total uint64
	files := make(map[string]*zip.File, len(zr.File))
	for _, file := range zr.File {
		total += file.UncompressedSize64
		if total > maxUncompressedBytes {
			return nil, fmt.Errorf("zip uncompressed size exceeds limit: %d > %d", total, maxUncompressedBytes)
		}
		name := cleanPartName(file.Name)
		files[name] = file
	}

	p := &Package{
		reader:       zr,
		files:        files,
		defaultTypes: make(map[string]string),
		overrides:    make(map[string]string),
	}
	if err := p.parseContentTypes(); err != nil {
		return nil, err
	}

	return p, nil
}

// MainDocumentPart returns the package path of the main Word document part.
func (p *Package) MainDocumentPart() (string, error) {
	for part, contentType := range p.overrides {
		if strings.HasSuffix(contentType, mainDocumentType) {
			return part, nil
		}
	}
	return "", errors.New("main document part not found")
}

// OpenPart opens a package part by name.
func (p *Package) OpenPart(name string) (io.ReadCloser, error) {
	file, ok := p.files[cleanPartName(name)]
	if !ok {
		return nil, fmt.Errorf("part %q not found", name)
	}
	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening part %q: %w", name, err)
	}
	return r, nil
}

// ReadXMLPart reads a bounded XML part into memory.
func (p *Package) ReadXMLPart(name string) ([]byte, error) {
	r, err := p.OpenPart(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	limited := io.LimitReader(r, maxXMLPartBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading XML part %q: %w", name, err)
	}
	if len(data) > maxXMLPartBytes {
		return nil, fmt.Errorf("XML part %q exceeds limit: %d > %d", name, len(data), maxXMLPartBytes)
	}
	return data, nil
}

// ContentType returns the content type for a package part.
func (p *Package) ContentType(name string) string {
	name = cleanPartName(name)
	if contentType, ok := p.overrides[name]; ok {
		return contentType
	}
	ext := strings.TrimPrefix(path.Ext(name), ".")
	return p.defaultTypes[strings.ToLower(ext)]
}

// Relationships reads the .rels part associated with sourcePart.
func (p *Package) Relationships(sourcePart string) (map[string]Relationship, error) {
	relsPart := RelationshipsPart(sourcePart)
	if _, ok := p.files[relsPart]; !ok {
		return map[string]Relationship{}, nil
	}

	data, err := p.ReadXMLPart(relsPart)
	if err != nil {
		return nil, err
	}
	rels, err := ParseRelationships(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing relationships %q: %w", relsPart, err)
	}
	for id, rel := range rels {
		if rel.TargetMode != "External" {
			rel.Target = ResolveTarget(sourcePart, rel.Target)
			rels[id] = rel
		}
	}
	return rels, nil
}

func (p *Package) parseContentTypes() error {
	data, err := p.ReadXMLPart(contentTypesPart)
	if err != nil {
		return err
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parsing content types: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "Default":
			ext := attr(se, "Extension")
			contentType := attr(se, "ContentType")
			if ext != "" && contentType != "" {
				p.defaultTypes[strings.ToLower(ext)] = contentType
			}
		case "Override":
			part := cleanPartName(attr(se, "PartName"))
			contentType := attr(se, "ContentType")
			if part != "" && contentType != "" {
				p.overrides[part] = contentType
			}
		}
	}
}

func attr(se xml.StartElement, local string) string {
	for _, attr := range se.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func cleanPartName(name string) string {
	name = strings.TrimPrefix(name, "/")
	clean := path.Clean(name)
	if clean == "." {
		return ""
	}
	return clean
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
