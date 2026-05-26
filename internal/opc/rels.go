package opc

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const (
	// ImageRelationshipType is the OOXML relationship type for embedded images.
	ImageRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	// HyperlinkRelationshipType is the OOXML relationship type for hyperlinks.
	HyperlinkRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"
	// NumberingRelationshipType is the OOXML relationship type for numbering definitions.
	NumberingRelationshipType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering"
)

// Relationship is one OPC relationship entry.
type Relationship struct {
	ID         string
	Type       string
	Target     string
	TargetMode string
}

// ParseRelationships parses an OPC .rels file into relationships keyed by ID.
func ParseRelationships(r io.Reader) (map[string]Relationship, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	rels := make(map[string]Relationship)

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return rels, nil
		}
		if err != nil {
			return nil, fmt.Errorf("reading relationship token: %w", err)
		}
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Relationship" {
			continue
		}

		rel := Relationship{
			ID:         attr(se, "Id"),
			Type:       attr(se, "Type"),
			Target:     attr(se, "Target"),
			TargetMode: attr(se, "TargetMode"),
		}
		if rel.ID != "" {
			rels[rel.ID] = rel
		}
	}
}

// RelationshipsPart returns the .rels part path for a source package part.
func RelationshipsPart(sourcePart string) string {
	sourcePart = cleanPartName(sourcePart)
	dir, base := path.Split(sourcePart)
	return path.Join(dir, "_rels", base+".rels")
}

// ResolveTarget resolves a relationship target relative to its source part.
func ResolveTarget(sourcePart, target string) string {
	if strings.HasPrefix(target, "/") {
		return cleanPartName(target)
	}
	dir, _ := path.Split(cleanPartName(sourcePart))
	return cleanPartName(path.Join(dir, target))
}

// XMLInt returns an integer attribute value with fallback.
func XMLInt(value string, fallback int) int {
	return parseInt(value, fallback)
}
