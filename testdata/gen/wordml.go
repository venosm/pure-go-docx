package main

import (
	"fmt"
	"strconv"
	"strings"
)

// text returns a run carrying literal text with whitespace preserved.
func text(value string) string {
	return `<w:r><w:t xml:space="preserve">` + escapeXML(value) + `</w:t></w:r>`
}

// styledText returns a run with the requested direct formatting flags.
func styledText(value string, bold, italic, underline bool) string {
	var props strings.Builder
	if bold {
		props.WriteString("<w:b/>")
	}
	if italic {
		props.WriteString("<w:i/>")
	}
	if underline {
		props.WriteString(`<w:u w:val="single"/>`)
	}

	run := "<w:r>"
	if props.Len() > 0 {
		run += "<w:rPr>" + props.String() + "</w:rPr>"
	}
	return run + `<w:t xml:space="preserve">` + escapeXML(value) + `</w:t></w:r>`
}

// tabRun returns a run containing a single tab character.
func tabRun() string {
	return "<w:r><w:tab/></w:r>"
}

// breakRun returns a run containing an explicit line break.
func breakRun() string {
	return "<w:r><w:br/></w:r>"
}

// noteRun returns a run referencing a footnote or endnote definition.
func noteRun(kind, id string) string {
	return `<w:r><w:rPr><w:rStyle w:val="FootnoteReference"/></w:rPr><w:` + kind + `Reference w:id="` + id + `"/></w:r>`
}

// paragraph joins runs into a paragraph without paragraph properties.
func paragraph(runs ...string) string {
	return "<w:p>" + strings.Join(runs, "") + "</w:p>"
}

// styledParagraph joins runs into a paragraph carrying a paragraph style.
func styledParagraph(styleID string, runs ...string) string {
	return `<w:p><w:pPr><w:pStyle w:val="` + styleID + `"/></w:pPr>` + strings.Join(runs, "") + "</w:p>"
}

// heading returns a paragraph using the built-in Heading<level> style.
func heading(level int, value string) string {
	return styledParagraph("Heading"+strconv.Itoa(level), text(value))
}

// listParagraph returns a list item bound to a numbering definition and level.
func listParagraph(numID, level int, runs ...string) string {
	return `<w:p><w:pPr><w:pStyle w:val="ListParagraph"/><w:numPr>` +
		`<w:ilvl w:val="` + strconv.Itoa(level) + `"/>` +
		`<w:numId w:val="` + strconv.Itoa(numID) + `"/>` +
		`</w:numPr></w:pPr>` + strings.Join(runs, "") + "</w:p>"
}

// hyperlink wraps runs in a relationship-based hyperlink.
func hyperlink(relID string, runs ...string) string {
	return `<w:hyperlink r:id="` + relID + `">` + strings.Join(runs, "") + "</w:hyperlink>"
}

// anchorLink wraps runs in an internal bookmark hyperlink.
func anchorLink(anchor string, runs ...string) string {
	return `<w:hyperlink w:anchor="` + escapeXML(anchor) + `">` + strings.Join(runs, "") + "</w:hyperlink>"
}

// complexField renders a begin/instruction/separate/result/end field sequence.
func complexField(instruction, result string) string {
	return `<w:r><w:fldChar w:fldCharType="begin"/></w:r>` +
		`<w:r><w:instrText xml:space="preserve">` + escapeXML(instruction) + `</w:instrText></w:r>` +
		`<w:r><w:fldChar w:fldCharType="separate"/></w:r>` +
		text(result) +
		`<w:r><w:fldChar w:fldCharType="end"/></w:r>`
}

// simpleField renders a w:fldSimple field with a cached display value.
func simpleField(instruction, result string) string {
	return `<w:fldSimple w:instr="` + escapeXML(instruction) + `">` + text(result) + "</w:fldSimple>"
}

// alternateContent wraps choice and fallback markup in an mc:AlternateContent block.
func alternateContent(requires, choiceXML, fallbackXML string) string {
	return `<mc:AlternateContent>` +
		`<mc:Choice Requires="` + requires + `">` + choiceXML + `</mc:Choice>` +
		`<mc:Fallback>` + fallbackXML + `</mc:Fallback>` +
		`</mc:AlternateContent>`
}

// structuredTag wraps blocks in a structured document tag (content control).
func structuredTag(alias, tag, contentXML string) string {
	return `<w:sdt><w:sdtPr><w:alias w:val="` + escapeXML(alias) + `"/><w:tag w:val="` + escapeXML(tag) + `"/>` +
		`<w:id w:val="123456789"/><w:text/></w:sdtPr>` +
		`<w:sdtContent>` + contentXML + `</w:sdtContent></w:sdt>`
}

// cell is one table cell description used by the table helper.
type cell struct {
	// blocks is the raw block XML stored inside the cell.
	blocks string
	// gridSpan is the horizontal span; zero and one mean a single column.
	gridSpan int
	// vMerge is empty, "restart", or "continue".
	vMerge string
}

// textCell returns a cell holding a single plain paragraph.
func textCell(value string) cell {
	return cell{blocks: paragraph(text(value))}
}

// boldCell returns a header cell holding a single bold paragraph.
func boldCell(value string) cell {
	return cell{blocks: paragraph(styledText(value, true, false, false))}
}

// table renders a table with an explicit grid of the given column count.
func table(columns int, rows [][]cell) string {
	var b strings.Builder
	b.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/></w:tblPr><w:tblGrid>`)
	for range columns {
		b.WriteString(`<w:gridCol w:w="3000"/>`)
	}
	b.WriteString("</w:tblGrid>")

	for _, row := range rows {
		b.WriteString("<w:tr>")
		for _, c := range row {
			b.WriteString("<w:tc>")
			b.WriteString(cellProperties(c))
			if c.blocks == "" {
				b.WriteString(paragraph())
			} else {
				b.WriteString(c.blocks)
			}
			b.WriteString("</w:tc>")
		}
		b.WriteString("</w:tr>")
	}
	b.WriteString("</w:tbl>")
	return b.String()
}

func cellProperties(c cell) string {
	var props strings.Builder
	if c.gridSpan > 1 {
		props.WriteString(`<w:gridSpan w:val="` + strconv.Itoa(c.gridSpan) + `"/>`)
	}
	switch c.vMerge {
	case "restart":
		props.WriteString(`<w:vMerge w:val="restart"/>`)
	case "continue":
		props.WriteString("<w:vMerge/>")
	}
	if props.Len() == 0 {
		return ""
	}
	return "<w:tcPr>" + props.String() + "</w:tcPr>"
}

// inlineDrawing renders an inline picture referencing an image relationship.
func inlineDrawing(id int, relID, name, altText string, widthEMU, heightEMU int64) string {
	extent := fmt.Sprintf(`<wp:extent cx="%d" cy="%d"/>`, widthEMU, heightEMU)
	docPr := fmt.Sprintf(`<wp:docPr id="%d" name="%s" descr="%s"/>`, id, escapeXML(name), escapeXML(altText))

	return `<w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0">` +
		extent +
		`<wp:effectExtent l="0" t="0" r="0" b="0"/>` +
		docPr +
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>` +
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture">` +
		`<pic:pic><pic:nvPicPr>` +
		fmt.Sprintf(`<pic:cNvPr id="%d" name="%s" descr="%s"/>`, id, escapeXML(name), escapeXML(altText)) +
		`<pic:cNvPicPr/></pic:nvPicPr>` +
		`<pic:blipFill><a:blip r:embed="` + relID + `"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>` +
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/>` +
		fmt.Sprintf(`<a:ext cx="%d" cy="%d"/>`, widthEMU, heightEMU) +
		`</a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr>` +
		`</pic:pic></a:graphicData></a:graphic>` +
		`</wp:inline></w:drawing>`
}

// imageParagraph returns a paragraph holding one inline picture.
func imageParagraph(id int, relID, name, altText string) string {
	return paragraph("<w:r>" + inlineDrawing(id, relID, name, altText, 914400, 914400) + "</w:r>")
}

// sectionProperties renders a sectPr with optional header and footer references.
func sectionProperties(refs ...string) string {
	return "<w:sectPr>" + strings.Join(refs, "") +
		`<w:pgSz w:w="11906" w:h="16838"/>` +
		`<w:pgMar w:top="1417" w:right="1417" w:bottom="1417" w:left="1417" w:header="708" w:footer="708" w:gutter="0"/>` +
		"</w:sectPr>"
}

// headerReference binds a header part relationship to a section page type.
func headerReference(pageType, relID string) string {
	return `<w:headerReference w:type="` + pageType + `" r:id="` + relID + `"/>`
}

// footerReference binds a footer part relationship to a section page type.
func footerReference(pageType, relID string) string {
	return `<w:footerReference w:type="` + pageType + `" r:id="` + relID + `"/>`
}
