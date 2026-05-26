package godocx

import (
	"fmt"
	"strconv"
	"strings"
)

// ToText returns a plain-text linearization of the document body.
func (d *Document) ToText() string {
	var b strings.Builder
	writeBlocksText(&b, d.Body)
	return b.String()
}

// ToMarkdown returns a markdown linearization suitable for embedding/indexing.
func (d *Document) ToMarkdown() string {
	var b strings.Builder
	writeBlocksMarkdown(&b, d.Body)
	return b.String()
}

// Chunks returns ordered chunks ready for RAG ingestion.
func (d *Document) Chunks() []Chunk {
	var chunks []Chunk
	tableID := 0
	appendChunks(&chunks, d.Body, &tableID)
	return chunks
}

func writeBlocksText(b *strings.Builder, blocks []Block) {
	for i, block := range blocks {
		switch value := block.(type) {
		case *Paragraph:
			b.WriteString(paragraphTextWithPrefix(value))
			b.WriteByte('\n')
		case *Table:
			if b.Len() > 0 {
				ensureBlankLine(b)
			}
			writeTableText(b, value)
			if i < len(blocks)-1 {
				b.WriteByte('\n')
			}
		}
	}
}

func writeTableText(b *strings.Builder, table *Table) {
	for _, row := range table.Grid {
		for i, cell := range row {
			if i > 0 {
				b.WriteByte('\t')
			}
			b.WriteString(tableCellText(cell, 1))
		}
		b.WriteByte('\n')
	}
}

func writeBlocksMarkdown(b *strings.Builder, blocks []Block) {
	for _, block := range blocks {
		switch value := block.(type) {
		case *Paragraph:
			writeParagraphMarkdown(b, value)
			b.WriteString("\n\n")
		case *Table:
			writeTableMarkdown(b, value)
			b.WriteByte('\n')
		}
	}
}

func writeParagraphMarkdown(b *strings.Builder, paragraph *Paragraph) {
	text := paragraphText(paragraph)
	if paragraph.HeadingLvl > 0 {
		b.WriteString(strings.Repeat("#", paragraph.HeadingLvl))
		b.WriteByte(' ')
		b.WriteString(text)
		return
	}
	if paragraph.List != nil {
		b.WriteString(strings.Repeat("  ", paragraph.List.Level))
		if paragraph.List.Format == "decimal" {
			b.WriteString(strconv.Itoa(paragraph.List.Ordinal))
			b.WriteString(". ")
		} else {
			b.WriteString("- ")
		}
	}
	b.WriteString(text)
}

// formatOrdinal renders n in the given OOXML numFmt. Unknown formats fall back to decimal.
func formatOrdinal(format string, n int) string {
	switch format {
	case "decimal":
		return strconv.Itoa(n)
	case "decimalZero":
		return fmt.Sprintf("%02d", n)
	case "lowerLetter":
		return formatLetterOrdinal(n, false)
	case "upperLetter":
		return formatLetterOrdinal(n, true)
	case "lowerRoman":
		return formatRomanOrdinal(n, false)
	case "upperRoman":
		return formatRomanOrdinal(n, true)
	default:
		return strconv.Itoa(n)
	}
}

func writeTableMarkdown(b *strings.Builder, table *Table) {
	if len(table.Grid) == 0 {
		return
	}
	writeMarkdownRow(b, table.Grid[0])
	writeMarkdownSeparator(b, len(table.Grid[0]))
	for _, row := range table.Grid[1:] {
		writeMarkdownRow(b, row)
	}
}

func writeMarkdownRow(b *strings.Builder, row []Cell) {
	b.WriteByte('|')
	for _, cell := range row {
		b.WriteByte(' ')
		b.WriteString(escapeMarkdownCell(cellText(cell)))
		b.WriteString(" |")
	}
	b.WriteByte('\n')
}

func writeMarkdownSeparator(b *strings.Builder, cols int) {
	b.WriteByte('|')
	for i := 0; i < cols; i++ {
		b.WriteString(" --- |")
	}
	b.WriteByte('\n')
}

func appendChunks(chunks *[]Chunk, blocks []Block, tableID *int) {
	for _, block := range blocks {
		switch value := block.(type) {
		case *Paragraph:
			text := paragraphText(value)
			kind := "paragraph"
			level := value.HeadingLvl
			if value.HeadingLvl > 0 {
				kind = "heading"
			}
			if value.List != nil {
				kind = "list-item"
				level = value.List.Level
			}
			if text != "" {
				*chunks = append(*chunks, Chunk{Kind: kind, Level: level, Text: text})
			}
			for _, run := range value.Runs {
				if run.Image != nil {
					*chunks = append(*chunks, Chunk{
						Kind:    "image",
						Text:    run.Image.AltText,
						ImageID: run.Image.RelID,
					})
				}
			}
		case *Table:
			*tableID++
			currentID := *tableID
			for _, row := range value.Grid {
				var b strings.Builder
				for i, cell := range row {
					if i > 0 {
						b.WriteByte('\t')
					}
					b.WriteString(cellText(cell))
				}
				*chunks = append(*chunks, Chunk{
					Kind:    "table-row",
					Text:    b.String(),
					TableID: currentID,
				})
				for _, cell := range row {
					appendChunks(chunks, cell.Blocks, tableID)
				}
			}
		}
	}
}

func paragraphText(paragraph *Paragraph) string {
	var b strings.Builder
	for _, run := range paragraph.Runs {
		writeRunText(&b, run)
	}
	return b.String()
}

func paragraphTextWithPrefix(paragraph *Paragraph) string {
	text := paragraphText(paragraph)
	if paragraph.List == nil {
		return text
	}

	var b strings.Builder
	b.WriteString(strings.Repeat("  ", paragraph.List.Level))
	switch paragraph.List.Format {
	case "bullet":
		b.WriteString("- ")
	case "none":
	default:
		b.WriteString(formatOrdinal(paragraph.List.Format, paragraph.List.Ordinal))
		b.WriteString(". ")
	}
	b.WriteString(text)
	return b.String()
}

func cellText(cell Cell) string {
	var b strings.Builder
	writeBlocksText(&b, cell.Blocks)
	return strings.TrimRight(b.String(), "\n")
}

func tableCellText(cell Cell, nestedIndent int) string {
	var parts []string
	hasNestedTable := false
	for _, block := range cell.Blocks {
		switch value := block.(type) {
		case *Paragraph:
			parts = append(parts, paragraphTextWithPrefix(value))
		case *Table:
			hasNestedTable = true
			parts = append(parts, nestedTableText(value, nestedIndent))
		}
	}

	text := strings.Join(parts, "\n")
	if hasNestedTable {
		// TODO: nested-table-in-row plaintext rendering is approximate.
		return text
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	return text
}

func nestedTableText(table *Table, indent int) string {
	var lines []string
	prefix := strings.Repeat("  ", indent)
	for _, row := range table.Grid {
		var b strings.Builder
		b.WriteString(prefix)
		for i, cell := range row {
			if i > 0 {
				b.WriteByte('\t')
			}
			b.WriteString(tableCellText(cell, indent+1))
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func ensureBlankLine(b *strings.Builder) {
	text := b.String()
	if strings.HasSuffix(text, "\n\n") {
		return
	}
	if !strings.HasSuffix(text, "\n") {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
}

func writeRunText(b *strings.Builder, run Run) {
	switch {
	case run.Image != nil:
		filename := run.Image.Filename
		if filename == "" {
			filename = run.Image.RelID
		}
		b.WriteString("[image: ")
		b.WriteString(filename)
		b.WriteByte(']')
	case run.Tab:
		b.WriteByte('\t')
	case run.Break:
		b.WriteByte('\n')
	default:
		b.WriteString(run.Text)
	}
}

func formatLetterOrdinal(n int, upper bool) string {
	if n <= 0 {
		return strconv.Itoa(n)
	}
	var chars []byte
	for n > 0 {
		n--
		chars = append(chars, byte('a'+n%26))
		n /= 26
	}
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	result := string(chars)
	if upper {
		return strings.ToUpper(result)
	}
	return result
}

func formatRomanOrdinal(n int, upper bool) string {
	if n <= 0 || n > 3999 {
		return strconv.Itoa(n)
	}
	values := []struct {
		value  int
		symbol string
	}{
		{1000, "M"},
		{900, "CM"},
		{500, "D"},
		{400, "CD"},
		{100, "C"},
		{90, "XC"},
		{50, "L"},
		{40, "XL"},
		{10, "X"},
		{9, "IX"},
		{5, "V"},
		{4, "IV"},
		{1, "I"},
	}

	var b strings.Builder
	for _, item := range values {
		for n >= item.value {
			b.WriteString(item.symbol)
			n -= item.value
		}
	}
	result := b.String()
	if upper {
		return result
	}
	return strings.ToLower(result)
}

func escapeMarkdownCell(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `|`, `\|`)
	text = strings.ReplaceAll(text, "\n", "<br>")
	return text
}
