package godocx

import (
	"fmt"
	"sort"
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
	for _, id := range sortedBlockKeys(d.Headers) {
		writeBlocksMarkdown(&b, d.Headers[id])
		ensureBlankLine(&b)
	}
	writeBlocksMarkdown(&b, d.Body)
	ensureBlankLine(&b)
	for _, id := range sortedBlockKeys(d.Footers) {
		writeBlocksMarkdown(&b, d.Footers[id])
		ensureBlankLine(&b)
	}
	writeNoteDefinitionsMarkdown(&b, NoteKindFootnote, d.Footnotes)
	writeNoteDefinitionsMarkdown(&b, NoteKindEndnote, d.Endnotes)
	result := strings.TrimRight(b.String(), "\n")
	if result == "" {
		return ""
	}
	return result + "\n"
}

// Chunks returns ordered chunks ready for RAG ingestion.
func (d *Document) Chunks() []Chunk {
	builder := chunkBuilder{}
	if pocet := d.odhadPoctuChunku(); pocet > 0 {
		builder.chunks = make([]Chunk, 0, pocet)
	}
	for _, id := range sortedBlockKeys(d.Headers) {
		builder.appendBlocks(d.Headers[id], chunkSource{kind: "header", id: id, path: "headers/" + id})
	}
	builder.appendBlocks(d.Body, chunkSource{kind: "body", path: "body"})
	for _, id := range sortedBlockKeys(d.Footers) {
		builder.appendBlocks(d.Footers[id], chunkSource{kind: "footer", id: id, path: "footers/" + id})
	}
	for _, id := range sortedNoteKeys(d.Footnotes) {
		builder.appendBlocks(d.Footnotes[id], chunkSource{kind: NoteKindFootnote, id: id, path: "footnotes/" + id})
	}
	for _, id := range sortedNoteKeys(d.Endnotes) {
		builder.appendBlocks(d.Endnotes[id], chunkSource{kind: NoteKindEndnote, id: id, path: "endnotes/" + id})
	}
	return builder.chunks
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
			markdown := paragraphMarkdown(value)
			if markdown == "" {
				continue
			}
			b.WriteString(markdown)
			b.WriteString("\n\n")
		case *Table:
			if b.Len() > 0 {
				ensureBlankLine(b)
			}
			writeTableMarkdown(b, value)
			b.WriteByte('\n')
		}
	}
}

func paragraphMarkdown(paragraph *Paragraph) string {
	var b strings.Builder
	b.Grow(delkaTextuBehu(paragraph.Runs) + 16)
	if paragraph.HeadingLvl > 0 {
		for range paragraph.HeadingLvl {
			b.WriteByte('#')
		}
		b.WriteByte(' ')
	} else if paragraph.List != nil {
		for range paragraph.List.Level {
			b.WriteString("  ")
		}
		if marker := listMarker(paragraph.List); marker != "" {
			b.WriteString(marker)
			b.WriteByte(' ')
		}
	}
	for _, run := range paragraph.Runs {
		writeRunMarkdown(&b, run)
	}
	return b.String()
}

func paragraphInlineMarkdown(paragraph *Paragraph) string {
	var b strings.Builder
	b.Grow(delkaTextuBehu(paragraph.Runs))
	for _, run := range paragraph.Runs {
		writeRunMarkdown(&b, run)
	}
	return b.String()
}

// delkaTextuBehu sečte délku textu běhů; slouží jako odhad pro Builder.Grow.
func delkaTextuBehu(runs []Run) int {
	delka := 0
	for _, run := range runs {
		delka += len(run.Text)
	}
	return delka
}

func listMarker(list *ListRef) string {
	if list == nil {
		return ""
	}
	switch list.Format {
	case "bullet":
		return "-"
	case "none":
		return ""
	default:
		return formatOrdinal(list.Format, list.Ordinal) + "."
	}
}

func writeNoteDefinitionsMarkdown(b *strings.Builder, kind string, notes map[string][]Block) {
	for _, id := range sortedNoteKeys(notes) {
		text := blocksMarkdownInline(notes[id])
		if text == "" {
			continue
		}
		ensureBlankLine(b)
		b.WriteString(noteMarkdownRef(NoteRef{Kind: kind, ID: id}))
		b.WriteString(": ")
		b.WriteString(text)
		b.WriteByte('\n')
	}
}

func blocksMarkdownInline(blocks []Block) string {
	var parts []string
	for _, block := range blocks {
		switch value := block.(type) {
		case *Paragraph:
			if text := paragraphInlineMarkdown(value); text != "" {
				parts = append(parts, text)
			}
		case *Table:
			if text := tablePlainText(value); text != "" {
				parts = append(parts, escapeMarkdownText(text))
			}
		}
	}
	return strings.Join(parts, " ")
}

func runMarkdown(run Run) string {
	var b strings.Builder
	writeRunMarkdown(&b, run)
	return b.String()
}

// writeRunMarkdown zapisuje běh přímo do b, bez mezilehlých řetězců
// pro každou vrstvu formátování. Pořadí značek odpovídá vnoření
// **, *, <u>, odkaz (od vnější po vnitřní).
func writeRunMarkdown(b *strings.Builder, run Run) {
	switch {
	case run.Image != nil:
		target := run.Image.Filename
		if target == "" {
			target = run.Image.ID
		}
		if target == "" {
			target = run.Image.RelID
		}
		b.WriteString("![")
		b.WriteString(escapeMarkdownText(run.Image.AltText))
		b.WriteString("](")
		b.WriteString(escapeMarkdownURL(target))
		b.WriteByte(')')
	case run.Note != nil:
		b.WriteString(noteMarkdownRef(*run.Note))
	case run.Tab:
		b.WriteByte('\t')
	case run.Break:
		b.WriteByte('\n')
	default:
		text := escapeMarkdownText(run.Text)
		if text == "" {
			return
		}
		if run.Bold {
			b.WriteString("**")
		}
		if run.Italic {
			b.WriteByte('*')
		}
		if run.Underline {
			b.WriteString("<u>")
		}
		if run.Link != "" {
			b.WriteByte('[')
		}
		b.WriteString(text)
		if run.Link != "" {
			b.WriteString("](")
			b.WriteString(escapeMarkdownURL(run.Link))
			b.WriteByte(')')
		}
		if run.Underline {
			b.WriteString("</u>")
		}
		if run.Italic {
			b.WriteByte('*')
		}
		if run.Bold {
			b.WriteString("**")
		}
	}
}

func noteMarkdownRef(note NoteRef) string {
	switch note.Kind {
	case NoteKindEndnote:
		return "[^en" + note.ID + "]"
	default:
		return "[^fn" + note.ID + "]"
	}
}

// znakyMarkdownu jsou znaky, které escapeMarkdownText escapuje.
const znakyMarkdownu = "\\*_[]`"

// nahrazovacMarkdownu se sestavuje jednou; strings.Replacer je bezpečný
// pro souběžné použití z více goroutin.
var nahrazovacMarkdownu = strings.NewReplacer(
	`\`, `\\`,
	`*`, `\*`,
	`_`, `\_`,
	`[`, `\[`,
	`]`, `\]`,
	"`", "\\`",
)

func escapeMarkdownText(text string) string {
	if !strings.ContainsAny(text, znakyMarkdownu) {
		return text
	}
	return nahrazovacMarkdownu.Replace(text)
}

func escapeMarkdownURL(url string) string {
	url = strings.ReplaceAll(url, `\`, `%5C`)
	url = strings.ReplaceAll(url, ")", "%29")
	url = strings.ReplaceAll(url, " ", "%20")
	return url
}

func tablePlainText(table *Table) string {
	var b strings.Builder
	writeTableText(&b, table)
	return strings.TrimRight(b.String(), "\n")
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
		b.WriteString(escapeMarkdownCell(cellMarkdown(cell)))
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

type chunkSource struct {
	kind string
	id   string
	path string
}

type chunkBuilder struct {
	chunks  []Chunk
	tableID int
}

// odhadPoctuChunku vrací horní mez počtu chunků ve všech částech dokumentu,
// aby Chunks alokoval výsledný slice jednou a append ho nemusel zvětšovat.
// Odstavce s běhy, ale bez textu, chunk nevytvoří, proto jde o horní mez,
// ne o přesný počet.
func (d *Document) odhadPoctuChunku() int {
	pocet := pocetChunkuBloku(d.Body)
	for _, casti := range []map[string][]Block{d.Headers, d.Footers, d.Footnotes, d.Endnotes} {
		for _, bloky := range casti {
			pocet += pocetChunkuBloku(bloky)
		}
	}
	return pocet
}

func pocetChunkuBloku(blocks []Block) int {
	pocet := 0
	for _, block := range blocks {
		switch value := block.(type) {
		case *Paragraph:
			// Odstavec bez běhů, který není nadpis ani položka seznamu,
			// má prázdný text i Markdown, a appendParagraph ho vynechá.
			if len(value.Runs) > 0 || value.HeadingLvl > 0 || value.List != nil {
				pocet++
			}
			for _, run := range value.Runs {
				if run.Image != nil {
					pocet++
				}
			}
		case *Table:
			for _, row := range value.Grid {
				pocet++
				for _, cell := range row {
					pocet += pocetChunkuBloku(cell.Blocks)
				}
			}
		}
	}
	return pocet
}

func (b *chunkBuilder) appendBlocks(blocks []Block, source chunkSource) {
	for i, block := range blocks {
		path := source.child(strconv.Itoa(i))
		switch value := block.(type) {
		case *Paragraph:
			b.appendParagraph(value, source, path)
		case *Table:
			b.appendTable(value, source, path)
		}
	}
}

func (b *chunkBuilder) appendParagraph(paragraph *Paragraph, source chunkSource, path string) {
	text := paragraphText(paragraph)
	markdown := paragraphMarkdown(paragraph)
	kind := "paragraph"
	level := paragraph.HeadingLvl
	if paragraph.HeadingLvl > 0 {
		kind = "heading"
	}
	var list *ChunkList
	if paragraph.List != nil {
		kind = "list-item"
		level = paragraph.List.Level
		list = &ChunkList{
			NumID:     paragraph.List.NumID,
			Level:     paragraph.List.Level,
			Format:    paragraph.List.Format,
			LevelText: paragraph.List.LevelText,
			Ordinal:   paragraph.List.Ordinal,
			Marker:    listMarker(paragraph.List),
		}
	}
	if text != "" || markdown != "" {
		b.appendChunk(Chunk{
			Kind:     kind,
			Level:    level,
			Text:     text,
			Markdown: markdown,
			StyleID:  paragraph.StyleID,
			List:     list,
			Notes:    paragraphNoteRefs(paragraph),
		}, source, path)
	}
	for runIndex, run := range paragraph.Runs {
		if run.Image == nil {
			continue
		}
		image := ChunkImage{
			ID:        run.Image.ID,
			PartName:  run.Image.PartName,
			RelID:     run.Image.RelID,
			Filename:  run.Image.Filename,
			AltText:   run.Image.AltText,
			WidthEMU:  run.Image.WidthEMU,
			HeightEMU: run.Image.HeightEMU,
		}
		imageID := image.ID
		if imageID == "" {
			imageID = run.Image.RelID
		}
		b.appendChunk(Chunk{
			Kind:     "image",
			Text:     run.Image.AltText,
			Markdown: runMarkdown(run),
			Image:    &image,
			ImageID:  imageID,
		}, source, path+"/image/"+strconv.Itoa(runIndex))
	}
}

func (b *chunkBuilder) appendTable(table *Table, source chunkSource, path string) {
	b.tableID++
	currentID := b.tableID
	for rowIndex, row := range table.Grid {
		text := rowText(row)
		markdown := rowMarkdown(row)
		b.appendChunk(Chunk{
			Kind:     "table-row",
			Text:     text,
			Markdown: markdown,
			Table: &ChunkTable{
				ID:      currentID,
				Row:     rowIndex,
				Column:  -1,
				Columns: len(row),
			},
			TableID: currentID,
		}, source, path+"/row/"+strconv.Itoa(rowIndex))
		for cellIndex, cell := range row {
			cellSource := source
			cellSource.path = path + "/row/" + strconv.Itoa(rowIndex) + "/cell/" + strconv.Itoa(cellIndex)
			b.appendBlocks(cell.Blocks, cellSource)
		}
	}
}

func (b *chunkBuilder) appendChunk(chunk Chunk, source chunkSource, path string) {
	chunk.Index = len(b.chunks)
	chunk.Source = source.kind
	chunk.SourceID = source.id
	chunk.Path = path
	chunk.ID = chunkID(source, path)
	b.chunks = append(b.chunks, chunk)
}

func (s chunkSource) child(part string) string {
	if s.path == "" {
		return part
	}
	return s.path + "/" + part
}

func chunkID(source chunkSource, path string) string {
	if source.id == "" {
		return source.kind + ":" + path
	}
	return source.kind + ":" + source.id + ":" + path
}

func paragraphNoteRefs(paragraph *Paragraph) []NoteRef {
	var notes []NoteRef
	for _, run := range paragraph.Runs {
		if run.Note == nil {
			continue
		}
		notes = append(notes, *run.Note)
	}
	return notes
}

func rowText(row []Cell) string {
	var b strings.Builder
	for i, cell := range row {
		if i > 0 {
			b.WriteByte('\t')
		}
		b.WriteString(cellText(cell))
	}
	return b.String()
}

func rowMarkdown(row []Cell) string {
	var b strings.Builder
	writeMarkdownRow(&b, row)
	return strings.TrimRight(b.String(), "\n")
}

func paragraphText(paragraph *Paragraph) string {
	var b strings.Builder
	b.Grow(delkaTextuBehu(paragraph.Runs))
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

func cellMarkdown(cell Cell) string {
	return blocksMarkdownInline(cell.Blocks)
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
		// Preserve nested table row breaks rather than flattening structure away.
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
	if b.Len() == 0 {
		return
	}
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
	case run.Note != nil:
		switch run.Note.Kind {
		case NoteKindEndnote:
			b.WriteString("[endnote: ")
		default:
			b.WriteString("[footnote: ")
		}
		b.WriteString(run.Note.ID)
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

func sortedBlockKeys(values map[string][]Block) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedNoteKeys(values map[string][]Block) []string {
	keys := sortedBlockKeys(values)
	sort.SliceStable(keys, func(i, j int) bool {
		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return keys[i] < keys[j]
	})
	return keys
}
