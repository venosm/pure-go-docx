package body

import (
	"encoding/xml"
	"strconv"
	"strings"
)

type runStyle struct {
	bold, italic, underline bool
}

type fieldState struct {
	depth       int
	showResult  bool
	instruction strings.Builder
	displayLink string
}

func (p *parser) parseParagraph() (*Paragraph, error) {
	paragraph := &Paragraph{}
	field := &fieldState{}
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
				runs, err := p.parseRun("", field)
				if err != nil {
					return nil, err
				}
				paragraph.Runs = append(paragraph.Runs, runs...)
			case "hyperlink":
				runs, err := p.parseHyperlink(t, field)
				if err != nil {
					return nil, err
				}
				paragraph.Runs = append(paragraph.Runs, runs...)
			case "fldSimple":
				runs, err := p.parseSimpleField("", t, field)
				if err != nil {
					return nil, err
				}
				paragraph.Runs = append(paragraph.Runs, runs...)
			case "AlternateContent":
				runs, err := p.parseAlternateContentRuns("", field)
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

func (p *parser) parseHyperlink(start xml.StartElement, field *fieldState) ([]Run, error) {
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
			switch t.Name.Local {
			case "r":
				parsed, err := p.parseRun(link, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "fldSimple":
				parsed, err := p.parseSimpleField(link, t, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "AlternateContent":
				parsed, err := p.parseAlternateContentRuns(link, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "hyperlink" {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseRun(link string, field *fieldState) ([]Run, error) {
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
				if field.shouldRender() {
					runs = append(runs, style.run(field.link(link), text))
				}
			case "tab":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Tab = true
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "br":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Break = true
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "drawing":
				image, err := p.parseDrawing()
				if err != nil {
					return nil, err
				}
				if image != nil && field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Image = image
					runs = append(runs, run)
				}
			case "footnoteReference":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Note = &NoteRef{Kind: NoteKindFootnote, ID: attr(t, "id")}
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "endnoteReference":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Note = &NoteRef{Kind: NoteKindEndnote, ID: attr(t, "id")}
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "fldChar":
				field.applyChar(attr(t, "fldCharType"))
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "instrText":
				text, err := p.readText("instrText")
				if err != nil {
					return nil, err
				}
				field.addInstruction(text)
			case "fldSimple":
				parsed, err := p.parseSimpleField(field.link(link), t, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "AlternateContent":
				parsed, err := p.parseAlternateContentRunChildren(field.link(link), &style, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
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

func (p *parser) parseRunsUntil(endLocal, link string, field *fieldState) ([]Run, error) {
	var runs []Run
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "r":
				parsed, err := p.parseRun(link, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "hyperlink":
				parsed, err := p.parseHyperlink(t, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "fldSimple":
				parsed, err := p.parseSimpleField(link, t, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "AlternateContent":
				parsed, err := p.parseAlternateContentRuns(link, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseSimpleField(link string, start xml.StartElement, parentField *fieldState) ([]Run, error) {
	if !parentField.shouldRender() {
		return nil, p.skipElement(start)
	}
	fieldLink := link
	if parsedLink := fieldInstructionLink(attr(start, "instr")); parsedLink != "" {
		fieldLink = parsedLink
	}
	return p.parseRunsUntil("fldSimple", fieldLink, &fieldState{})
}

func (p *parser) parseAlternateContentRuns(link string, field *fieldState) ([]Run, error) {
	var runs []Run
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
				choiceRuns, err := p.parseRunsUntil("Choice", link, field)
				if err != nil {
					return nil, err
				}
				runs = choiceRuns
				chosen = true
			case "Fallback":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return nil, err
					}
					continue
				}
				fallbackRuns, err := p.parseRunsUntil("Fallback", link, field)
				if err != nil {
					return nil, err
				}
				runs = fallbackRuns
				chosen = true
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "AlternateContent" {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseAlternateContentRunChildren(link string, style *runStyle, field *fieldState) ([]Run, error) {
	var runs []Run
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
				choiceRuns, err := p.parseRunChildrenUntil("Choice", link, style, field)
				if err != nil {
					return nil, err
				}
				runs = choiceRuns
				chosen = true
			case "Fallback":
				if chosen {
					if err := p.skipElement(t); err != nil {
						return nil, err
					}
					continue
				}
				fallbackRuns, err := p.parseRunChildrenUntil("Fallback", link, style, field)
				if err != nil {
					return nil, err
				}
				runs = fallbackRuns
				chosen = true
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == "AlternateContent" {
				return runs, nil
			}
		}
	}
}

func (p *parser) parseRunChildrenUntil(endLocal, link string, style *runStyle, field *fieldState) ([]Run, error) {
	var runs []Run
	for {
		tok, err := p.next()
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "r":
				parsed, err := p.parseRun(field.link(link), field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "rPr":
				if err := p.parseRunProperties(style); err != nil {
					return nil, err
				}
			case "t":
				text, err := p.readText("t")
				if err != nil {
					return nil, err
				}
				if field.shouldRender() {
					runs = append(runs, style.run(field.link(link), text))
				}
			case "tab":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Tab = true
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "br":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Break = true
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "drawing":
				image, err := p.parseDrawing()
				if err != nil {
					return nil, err
				}
				if image != nil && field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Image = image
					runs = append(runs, run)
				}
			case "footnoteReference":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Note = &NoteRef{Kind: NoteKindFootnote, ID: attr(t, "id")}
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "endnoteReference":
				if field.shouldRender() {
					run := style.run(field.link(link), "")
					run.Note = &NoteRef{Kind: NoteKindEndnote, ID: attr(t, "id")}
					runs = append(runs, run)
				}
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "fldChar":
				field.applyChar(attr(t, "fldCharType"))
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			case "instrText":
				text, err := p.readText("instrText")
				if err != nil {
					return nil, err
				}
				field.addInstruction(text)
			case "fldSimple":
				parsed, err := p.parseSimpleField(field.link(link), t, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			case "AlternateContent":
				parsed, err := p.parseAlternateContentRunChildren(field.link(link), style, field)
				if err != nil {
					return nil, err
				}
				runs = append(runs, parsed...)
			default:
				if err := p.skipElement(t); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			if t.Name.Local == endLocal {
				return runs, nil
			}
		}
	}
}

func (f *fieldState) shouldRender() bool {
	return f == nil || f.depth == 0 || f.showResult
}

func (f *fieldState) link(fallback string) string {
	if f == nil || f.displayLink == "" || !f.showResult {
		return fallback
	}
	return f.displayLink
}

func (f *fieldState) applyChar(charType string) {
	if f == nil {
		return
	}
	switch charType {
	case "begin":
		if f.depth == 0 {
			f.reset()
		}
		f.depth++
		f.showResult = false
	case "separate":
		if f.depth == 0 {
			return
		}
		f.showResult = true
		f.displayLink = fieldInstructionLink(f.instruction.String())
	case "end":
		if f.depth == 0 {
			return
		}
		f.depth--
		if f.depth == 0 {
			f.reset()
		}
	}
}

func (f *fieldState) addInstruction(text string) {
	if f == nil || f.depth == 0 || f.showResult {
		return
	}
	f.instruction.WriteString(text)
}

func (f *fieldState) reset() {
	f.showResult = false
	f.instruction.Reset()
	f.displayLink = ""
}

func fieldInstructionLink(instruction string) string {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return ""
	}
	fields := strings.Fields(instruction)
	if len(fields) == 0 || strings.ToUpper(fields[0]) != "HYPERLINK" {
		return ""
	}
	if start := strings.Index(instruction, `"`); start >= 0 {
		rest := instruction[start+1:]
		if end := strings.Index(rest, `"`); end >= 0 {
			return rest[:end]
		}
	}
	if len(fields) < 2 || strings.HasPrefix(fields[1], `\`) {
		return ""
	}
	return fields[1]
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
