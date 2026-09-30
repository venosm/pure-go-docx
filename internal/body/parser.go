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

	// otevrene je zásobník otevřených elementů s nepřeloženými prefixy.
	// Parser čte přes Decoder.RawToken a párování elementů kontroluje sám.
	otevrene []xml.StartElement
	// cekajiciKonec drží koncový tag, který se po automatickém uzavření
	// vnitřního elementu zpracuje znovu (nestriktní režim encoding/xml).
	cekajiciKonec   xml.EndElement
	maCekajiciKonec bool
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
		if space := p.jmennyProstor(se); space != "" && space != wordprocessingNamespace {
			return fmt.Errorf("unexpected %s namespace %q", rootLocal, space)
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

	tok, err := p.dalsiSurovyToken()
	if err != nil {
		return nil, err
	}

	switch t := tok.(type) {
	case xml.StartElement:
		p.depth++
		if p.depth > maxXMLDepth {
			return nil, fmt.Errorf("XML depth exceeds limit: %d > %d", p.depth, maxXMLDepth)
		}
		p.otevrene = append(p.otevrene, t)
	case xml.EndElement:
		if tok, err = p.uzavriElement(t); err != nil {
			return nil, err
		}
		if p.depth > 0 {
			p.depth--
		}
	}
	return tok, nil
}

// dalsiSurovyToken čte přes Decoder.RawToken, který na rozdíl od Token
// nepřekládá prefixy jmenných prostorů. Parser porovnává jen lokální jména,
// takže překlad by byl zbytečná práce. Konec vstupu uvnitř elementu hlásí
// stejnou chybou jako Token.
func (p *parser) dalsiSurovyToken() (xml.Token, error) {
	if p.maCekajiciKonec {
		p.maCekajiciKonec = false
		return p.cekajiciKonec, nil
	}
	tok, err := p.dec.RawToken()
	if tok == nil && err != nil {
		if errors.Is(err, io.EOF) && len(p.otevrene) > 0 {
			return nil, p.syntaktickaChyba("unexpected EOF")
		}
		return nil, err
	}
	return tok, nil
}

// uzavriElement páruje koncový tag s otevřeným elementem stejně jako
// Decoder.Token v nestriktním režimu: jiné lokální jméno uzavře vnitřní
// element a koncový tag se zpracuje znovu, jiný prefix je chyba.
func (p *parser) uzavriElement(konec xml.EndElement) (xml.Token, error) {
	n := len(p.otevrene)
	if n == 0 {
		return nil, p.syntaktickaChyba("unexpected end element </" + konec.Name.Local + ">")
	}
	zacatek := p.otevrene[n-1].Name
	p.otevrene = p.otevrene[:n-1]

	switch {
	case zacatek.Local != konec.Name.Local:
		p.cekajiciKonec = konec
		p.maCekajiciKonec = true
		return xml.EndElement{Name: zacatek}, nil
	case zacatek.Space != konec.Name.Space:
		space := konec.Name.Space
		if space == "" {
			space = `""`
		}
		return nil, p.syntaktickaChyba("element <" + zacatek.Local + "> in space " + zacatek.Space +
			" closed by </" + konec.Name.Local + "> in space " + space)
	}
	return konec, nil
}

// jmennyProstor přeloží prefix elementu na URI podle deklarací xmlns
// na něm a na jeho otevřených předcích, stejně jako Decoder.Token.
// Element už musí být na zásobníku otevrene.
func (p *parser) jmennyProstor(se xml.StartElement) string {
	prefix := se.Name.Space
	if prefix == "xml" {
		return "http://www.w3.org/XML/1998/namespace"
	}
	for i := len(p.otevrene) - 1; i >= 0; i-- {
		space, nalezeno := "", false
		for _, a := range p.otevrene[i].Attr {
			if (prefix == "" && a.Name.Space == "" && a.Name.Local == "xmlns") ||
				(prefix != "" && a.Name.Space == "xmlns" && a.Name.Local == prefix) {
				space, nalezeno = a.Value, true
			}
		}
		if nalezeno {
			return space
		}
	}
	return prefix
}

func (p *parser) syntaktickaChyba(zprava string) error {
	radek, _ := p.dec.InputPos()
	return &xml.SyntaxError{Msg: zprava, Line: radek}
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
