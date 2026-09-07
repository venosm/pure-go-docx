package body

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/venosm/pure-go-docx/internal/numbering"
	"github.com/venosm/pure-go-docx/internal/opc"
)

const (
	wordprocessingNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	maxXMLDepth             = 256
)

type parser struct {
	ctx           context.Context
	dec           *xml.Decoder
	depth         int
	relationships map[string]opc.Relationship
	numbering     *numbering.Resolver
	sourcePart    string
}

// Parse reads the main WordprocessingML document body into blocks.
func Parse(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver) ([]Block, error) {
	return ParseDocument(ctx, r, relationships, resolver, "")
}

// ParseDocument reads a WordprocessingML main document part into blocks.
func ParseDocument(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) ([]Block, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	p := &parser{
		ctx:           ctx,
		dec:           dec,
		relationships: relationships,
		numbering:     resolver,
		sourcePart:    sourcePart,
	}
	if err := p.seekRoot("document"); err != nil {
		return nil, err
	}
	return p.parseDocument()
}

// ParseHeader reads a WordprocessingML header part into blocks.
func ParseHeader(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) ([]Block, error) {
	return parseRootBlocks(ctx, r, relationships, resolver, sourcePart, "hdr")
}

// ParseFooter reads a WordprocessingML footer part into blocks.
func ParseFooter(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) ([]Block, error) {
	return parseRootBlocks(ctx, r, relationships, resolver, sourcePart, "ftr")
}

// ParseFootnotes reads a WordprocessingML footnotes part into blocks keyed by note ID.
func ParseFootnotes(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) (map[string][]Block, error) {
	return parseNotes(ctx, r, relationships, resolver, sourcePart, "footnotes", "footnote")
}

// ParseEndnotes reads a WordprocessingML endnotes part into blocks keyed by note ID.
func ParseEndnotes(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) (map[string][]Block, error) {
	return parseNotes(ctx, r, relationships, resolver, sourcePart, "endnotes", "endnote")
}

func newParser(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart string) *parser {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	return &parser{
		ctx:           ctx,
		dec:           dec,
		relationships: relationships,
		numbering:     resolver,
		sourcePart:    sourcePart,
	}
}

func parseRootBlocks(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart, rootLocal string) ([]Block, error) {
	p := newParser(ctx, r, relationships, resolver, sourcePart)
	if err := p.seekRoot(rootLocal); err != nil {
		return nil, err
	}
	return p.parseBlocksUntil(rootLocal)
}

func parseNotes(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver, sourcePart, rootLocal, noteLocal string) (map[string][]Block, error) {
	p := newParser(ctx, r, relationships, resolver, sourcePart)
	if err := p.seekRoot(rootLocal); err != nil {
		return nil, err
	}
	return p.parseNotes(rootLocal, noteLocal)
}

func (p *parser) seekRoot(rootLocal string) error {
	for {
		tok, err := p.next()
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%s root not found", rootLocal)
		}
		if err != nil {
			return err
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != rootLocal {
			continue
		}
		if se.Name.Space != "" && se.Name.Space != wordprocessingNamespace {
			return fmt.Errorf("unexpected %s namespace %q", rootLocal, se.Name.Space)
		}
		return nil
	}
}

func (p *parser) parseDocument() ([]Block, error) {
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "body" {
				return p.parseBlocksUntil("body")
			}
			if err := p.skipElement(t); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name.Local == "document" {
				return nil, errors.New("document body not found")
			}
		}
	}
}

func (p *parser) parseBlocksUntil(endLocal string) ([]Block, error) {
	var blocks []Block
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				paragraph, err := p.parseParagraph()
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, paragraph)
			case "tbl":
				table, err := p.parseTable()
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, table)
			case "sdt":
				sdtBlocks, err := p.parseSDT()
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, sdtBlocks...)
			case "AlternateContent":
				alternateBlocks, err := p.parseAlternateContentBlocks()
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, alternateBlocks...)
			case "sectPr":
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return blocks, nil
			}
		}
	}
}

func (p *parser) parseNotes(rootLocal, noteLocal string) (map[string][]Block, error) {
	notes := make(map[string][]Block)
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != noteLocal {
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
				continue
			}

			id := attr(t, "id")
			if id == "" || strings.HasPrefix(id, "-") || ignoredNoteType(attr(t, "type")) {
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
				continue
			}
			blocks, err := p.parseBlocksUntil(noteLocal)
			if err != nil {
				return nil, err
			}
			notes[id] = blocks
		case xml.EndElement:
			if t.Name.Local == rootLocal {
				return notes, nil
			}
		}
	}
}

func (p *parser) parseAlternateContentBlocks() ([]Block, error) {
	var blocks []Block
	chosen := false
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "Choice":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return nil, err
					}
					continue
				}
				choiceBlocks, err := p.parseBlocksUntil("Choice")
				if err != nil {
					return nil, err
				}
				blocks = choiceBlocks
				chosen = true
			case "Fallback":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return nil, err
					}
					continue
				}
				fallbackBlocks, err := p.parseBlocksUntil("Fallback")
				if err != nil {
					return nil, err
				}
				blocks = fallbackBlocks
				chosen = true
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "AlternateContent" {
				return blocks, nil
			}
		}
	}
}

func (p *parser) parseSDT() ([]Block, error) {
	var blocks []Block
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "sdtContent" {
				content, err := p.parseBlocksUntil("sdtContent")
				if err != nil {
					return nil, err
				}
				blocks = append(blocks, content...)
				continue
			}
			if err := p.skipElement(t); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name.Local == "sdt" {
				return blocks, nil
			}
		}
	}
}

func (p *parser) skipElement(start xml.StartElement) error {
	depth := 1
	for depth > 0 {
		tok, err := p.next()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == start.Name.Local {
				depth++
			}
		case xml.EndElement:
			if t.Name.Local == start.Name.Local {
				depth--
			}
		}
	}
	return nil
}

func (p *parser) next() (xml.Token, error) {
	select {
	case <-p.ctx.Done():
		return nil, p.ctx.Err()
	default:
	}

	tok, err := p.dec.Token()
	if err != nil {
		return nil, err
	}

	switch tok.(type) {
	case xml.StartElement:
		p.depth++
		if p.depth > maxXMLDepth {
			return nil, fmt.Errorf("XML depth exceeds limit: %d > %d", p.depth, maxXMLDepth)
		}
	case xml.EndElement:
		if p.depth > 0 {
			p.depth--
		}
	}
	return tok, nil
}

func attr(se xml.StartElement, local string) string {
	for _, attr := range se.Attr {
		if attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func ignoredNoteType(noteType string) bool {
	switch noteType {
	case "separator", "continuationSeparator", "continuationNotice":
		return true
	default:
		return false
	}
}
