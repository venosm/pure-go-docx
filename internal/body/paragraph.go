package body

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type runStyle struct {
	bold, italic, underline bool
}

func (p *parser) parseParagraph() (*Paragraph, error) {
	paragraph := &Paragraph{}
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "pPr":
				if err := p.parseParagraphProperties(paragraph); err != nil {
					return nil, err
				}
			case "r":
				runs, err := p.parseRun("")
				if err != nil {
					return nil, err
				}
				paragraph.Runs = append(paragraph.Runs, runs...)
			case "hyperlink":
				runs, err := p.parseHyperlink(t)
				if err != nil {
					return nil, err
				}
				paragraph.Runs = append(paragraph.Runs, runs...)
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "p" {
				return paragraph, nil
			}
		}
	}
}

func (p *parser) parseParagraphProperties(paragraph *Paragraph) error {
	for {
		tok, err := p.next()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "pStyle":
				paragraph.StyleID = attr(t, "val")
				paragraph.HeadingLvl = headingLevel(paragraph.StyleID)
			case "numPr":
				numID, level, err := p.parseNumPr()
				if err != nil {
					return err
				}
				if numID > 0 && p.numbering != nil {
					if def, ordinal, ok := p.numbering.Resolve(numID, level); ok {
						paragraph.List = &ListRef{
							NumID:     numID,
							Level:     level,
							Format:    def.Format,
							LevelText: def.LevelText,
							Ordinal:   ordinal,
						}
					}
				}
			default:
				if err := p.skipElement(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "pPr" {
				return nil
			}
		}
	}
}

func (p *parser) parseNumPr() (int, int, error) {
	numID := 0
	level := 0
	for {
		tok, err := p.next()
		if err != nil {
			return 0, 0, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "numId":
				numID = xmlInt(attr(t, "val"), 0)
				if err := p.skipElement(t); err != nil {
					return 0, 0, err
				}
			case "ilvl":
				level = xmlInt(attr(t, "val"), 0)
				if err := p.skipElement(t); err != nil {
					return 0, 0, err
				}
			default:
				if err := p.skipElement(t); err != nil {
					return 0, 0, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "numPr" {
				return numID, level, nil
			}
		}
	}
}

func (p *parser) parseHyperlink(start xml.StartElement) ([]Run, error) {
	link := ""
	if relID := attr(start, "id"); relID != "" {
		if rel, ok := p.relationships[relID]; ok {
			link = rel.Target
		}
	}
	if anchor := attr(start, "anchor"); link == "" && anchor != "" {
		link = "#" + anchor
	}

	var runs []Run
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "r" {
				parsed, err := p.parseRun(link)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
				continue
			}
			if err := p.skipElement(t); err != nil {
				return nil, err
			}
		case xml.EndElement:
			if t.Name.Local == "hyperlink" {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseRun(link string) ([]Run, error) {
	var runs []Run
	style := runStyle{}

	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "rPr":
				if err := p.parseRunProperties(&style); err != nil {
					return nil, err
				}
			case "t":
				text, err := p.readText("t")
				if err != nil {
					return nil, err
				}
				runs = append(runs, style.run(link, text))
			case "tab":
				run := style.run(link, "")
				run.Tab = true
				runs = append(runs, run)
			case "br":
				run := style.run(link, "")
				run.Break = true
				runs = append(runs, run)
			case "drawing":
				image, err := p.parseDrawing()
				if err != nil {
					return nil, err
				}
				if image != nil {
					run := style.run(link, "")
					run.Image = image
					runs = append(runs, run)
				}
			case "instrText":
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "r" {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseRunProperties(style *runStyle) error {
	for {
		tok, err := p.next()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "b":
				style.bold = !falseValue(attr(t, "val"))
			case "i":
				style.italic = !falseValue(attr(t, "val"))
			case "u":
				style.underline = !underlineNone(attr(t, "val"))
			default:
				if err := p.skipElement(t); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "rPr" {
				return nil
			}
		}
	}
}

func (p *parser) readText(endLocal string) (string, error) {
	var b strings.Builder
	for {
		tok, err := p.next()
		if err != nil {
			return "", err
		}

		switch t := tok.(type) {
		case xml.CharData:
			b.Write([]byte(t))
		case xml.StartElement:
			if err := p.skipElement(t); err != nil {
				return "", err
			}
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return b.String(), nil
			}
		}
	}
}

func (s runStyle) run(link, text string) Run {
	return Run{
		Text:      text,
		Bold:      s.bold,
		Italic:    s.italic,
		Underline: s.underline,
		Link:      link,
	}
}

func headingLevel(styleID string) int {
	for _, prefix := range []string{"Heading", "Nadpis"} {
		if !strings.HasPrefix(styleID, prefix) {
			continue
		}
		value := strings.TrimPrefix(styleID, prefix)
		level, err := strconv.Atoi(value)
		if err == nil && level >= 1 && level <= 9 {
			return level
		}
	}
	return 0
}

func xmlInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func falseValue(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "off":
		return true
	default:
		return false
	}
}

func underlineNone(value string) bool {
	return strings.EqualFold(value, "none") || falseValue(value)
}
