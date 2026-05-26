package body

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

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
}

// Parse reads the main WordprocessingML document body into blocks.
func Parse(ctx context.Context, r io.Reader, relationships map[string]opc.Relationship, resolver *numbering.Resolver) ([]Block, error) {
	dec := xml.NewDecoder(r)
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	p := &parser{
		ctx:           ctx,
		dec:           dec,
		relationships: relationships,
		numbering:     resolver,
	}

	for {
		tok, err := p.next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("document root not found")
		}
		if err != nil {
			return nil, err
		}

		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "document" {
			continue
		}
		if se.Name.Space != wordprocessingNamespace {
			return nil, fmt.Errorf("unexpected document namespace %q", se.Name.Space)
		}
		return p.parseDocument()
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
