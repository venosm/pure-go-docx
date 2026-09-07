package godocx

import (
	"bytes"
	"testing"

	"github.com/venosm/pure-go-docx/internal/testutil"
)

func TestFormatOrdinal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format string
		n      int
		want   string
	}{
		{name: "decimal 1", format: "decimal", n: 1, want: "1"},
		{name: "decimal 10", format: "decimal", n: 10, want: "10"},
		{name: "decimal 100", format: "decimal", n: 100, want: "100"},
		{name: "decimalZero 1", format: "decimalZero", n: 1, want: "01"},
		{name: "decimalZero 10", format: "decimalZero", n: 10, want: "10"},
		{name: "decimalZero 100", format: "decimalZero", n: 100, want: "100"},
		{name: "lowerLetter 1", format: "lowerLetter", n: 1, want: "a"},
		{name: "lowerLetter 26", format: "lowerLetter", n: 26, want: "z"},
		{name: "lowerLetter 27", format: "lowerLetter", n: 27, want: "aa"},
		{name: "lowerLetter 52", format: "lowerLetter", n: 52, want: "az"},
		{name: "lowerLetter 53", format: "lowerLetter", n: 53, want: "ba"},
		{name: "upperLetter 1", format: "upperLetter", n: 1, want: "A"},
		{name: "upperLetter 26", format: "upperLetter", n: 26, want: "Z"},
		{name: "upperLetter 27", format: "upperLetter", n: 27, want: "AA"},
		{name: "upperLetter 52", format: "upperLetter", n: 52, want: "AZ"},
		{name: "upperLetter 53", format: "upperLetter", n: 53, want: "BA"},
		{name: "lowerRoman 1", format: "lowerRoman", n: 1, want: "i"},
		{name: "lowerRoman 4", format: "lowerRoman", n: 4, want: "iv"},
		{name: "lowerRoman 9", format: "lowerRoman", n: 9, want: "ix"},
		{name: "lowerRoman 40", format: "lowerRoman", n: 40, want: "xl"},
		{name: "lowerRoman 49", format: "lowerRoman", n: 49, want: "xlix"},
		{name: "lowerRoman 90", format: "lowerRoman", n: 90, want: "xc"},
		{name: "lowerRoman 400", format: "lowerRoman", n: 400, want: "cd"},
		{name: "lowerRoman 900", format: "lowerRoman", n: 900, want: "cm"},
		{name: "lowerRoman 1994", format: "lowerRoman", n: 1994, want: "mcmxciv"},
		{name: "upperRoman 1", format: "upperRoman", n: 1, want: "I"},
		{name: "upperRoman 4", format: "upperRoman", n: 4, want: "IV"},
		{name: "upperRoman 9", format: "upperRoman", n: 9, want: "IX"},
		{name: "upperRoman 40", format: "upperRoman", n: 40, want: "XL"},
		{name: "upperRoman 49", format: "upperRoman", n: 49, want: "XLIX"},
		{name: "upperRoman 90", format: "upperRoman", n: 90, want: "XC"},
		{name: "upperRoman 400", format: "upperRoman", n: 400, want: "CD"},
		{name: "upperRoman 900", format: "upperRoman", n: 900, want: "CM"},
		{name: "upperRoman 1994", format: "upperRoman", n: 1994, want: "MCMXCIV"},
		{name: "unknown", format: "foobar", n: 12, want: "12"},
		{name: "roman zero", format: "lowerRoman", n: 0, want: "0"},
		{name: "roman too large", format: "upperRoman", n: 4000, want: "4000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := formatOrdinal(tc.format, tc.n); got != tc.want {
				t.Fatalf("formatOrdinal(%q, %d) = %q, want %q", tc.format, tc.n, got, tc.want)
			}
		})
	}
}

func TestToText_BulletList(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>first</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>nested</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>third</w:t></w:r></w:p>`, bulletNumberingXML())

	if got, want := doc.ToText(), "- first\n  - nested\n- third\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToText_NumberedList_Nested(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>first</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>nested a</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>nested b</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>second</w:t></w:r></w:p>`, nestedNumberingXML())

	if got, want := doc.ToText(), "1. first\n  a. nested a\n  b. nested b\n2. second\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToText_Table_FlatGrid(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>`+tc("A")+tc("B")+`</w:tr>
  <w:tr>`+tc("C")+tc("D")+`</w:tr>
</w:tbl>`)

	if got, want := doc.ToText(), "A\tB\nC\tD\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToText_Table_WithVerticalMerge(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    <w:tc><w:tcPr><w:vMerge w:val="restart"/></w:tcPr><w:p><w:r><w:t>Smlouva</w:t></w:r></w:p></w:tc>
    `+tc("42")+`
  </w:tr>
  <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>`+tc("43")+`</w:tr>
  <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr></w:tc>`+tc("44")+`</w:tr>
</w:tbl>`)

	if got, want := doc.ToText(), "Smlouva\t42\nSmlouva\t43\nSmlouva\t44\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToText_Table_WithHorizontalMerge(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>
    <w:tc><w:tcPr><w:gridSpan w:val="3"/></w:tcPr><w:p><w:r><w:t>span</w:t></w:r></w:p></w:tc>
    `+tc("B")+`
  </w:tr>
</w:tbl>`)

	if got, want := doc.ToText(), "span\t\t\tB\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToText_MixedBlocks(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<w:p><w:r><w:t>before</w:t></w:r></w:p>
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>`+tc("A")+tc("B")+`</w:tr>
</w:tbl>
<w:p><w:r><w:t>after</w:t></w:r></w:p>`)

	if got, want := doc.ToText(), "before\n\nA\tB\n\nafter\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestToMarkdown_RelatedPartsNotesAndFormatting(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Title</w:t></w:r>
</w:p>
<w:p>
  <w:r><w:rPr><w:b/></w:rPr><w:t>Body</w:t></w:r>
  <w:r><w:t xml:space="preserve"> with </w:t></w:r>
  <w:hyperlink r:id="rIdLink"><w:r><w:t>link</w:t></w:r></w:hyperlink>
  <w:r><w:footnoteReference w:id="2"/></w:r>
</w:p>`, `
<Relationship Id="rIdHeader" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
<Relationship Id="rIdFooter" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>
<Relationship Id="rIdFootnotes" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes" Target="footnotes.xml"/>
<Relationship Id="rIdLink" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="https://example.com" TargetMode="External"/>`,
		testutil.File{Name: "word/header1.xml", Data: testutil.WordPartXML("hdr", `<w:p><w:r><w:t>Header</w:t></w:r></w:p>`)},
		testutil.File{Name: "word/footer1.xml", Data: testutil.WordPartXML("ftr", `<w:p><w:r><w:t>Footer</w:t></w:r></w:p>`)},
		testutil.File{Name: "word/footnotes.xml", Data: testutil.WordPartXML("footnotes", `<w:footnote w:id="2"><w:p><w:r><w:t>Footnote text</w:t></w:r></w:p></w:footnote>`)},
	)
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	want := "Header\n\n# Title\n\n**Body** with [link](https://example.com)[^fn2]\n\nFooter\n\n[^fn2]: Footnote text\n"
	if got := doc.ToMarkdown(); got != want {
		t.Fatalf("ToMarkdown() = %q, want %q", got, want)
	}
}

func TestChunks_MetadataListAndTable(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p>
  <w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr>
  <w:r><w:t>first</w:t></w:r>
</w:p>
<w:tbl>
  <w:tblGrid><w:gridCol/><w:gridCol/></w:tblGrid>
  <w:tr>`+tc("A")+tc("B")+`</w:tr>
</w:tbl>`, decimalNumberingXML())

	chunks := doc.Chunks()
	if len(chunks) != 4 {
		t.Fatalf("len(Chunks()) = %d, want 4: %#v", len(chunks), chunks)
	}
	list := chunks[0]
	if list.ID != "body:body/0" || list.Index != 0 || list.Source != "body" || list.Path != "body/0" {
		t.Fatalf("list chunk identity = %#v", list)
	}
	if list.Kind != "list-item" || list.Level != 0 || list.Text != "first" || list.Markdown != "1. first" {
		t.Fatalf("list chunk content = %#v", list)
	}
	if list.List == nil || list.List.Marker != "1." || list.List.Ordinal != 1 || list.List.Format != "decimal" {
		t.Fatalf("list metadata = %#v, want decimal marker", list.List)
	}

	row := chunks[1]
	if row.Kind != "table-row" || row.Text != "A\tB" || row.Markdown != "| A | B |" {
		t.Fatalf("row chunk content = %#v", row)
	}
	if row.Table == nil || row.Table.ID != 1 || row.Table.Row != 0 || row.Table.Column != -1 || row.Table.Columns != 2 {
		t.Fatalf("row table metadata = %#v", row.Table)
	}
	if row.TableID != 1 {
		t.Fatalf("row.TableID = %d, want 1", row.TableID)
	}
	if chunks[2].Path != "body/1/row/0/cell/0/0" || chunks[2].Text != "A" {
		t.Fatalf("cell 0 nested paragraph chunk = %#v", chunks[2])
	}
	if chunks[3].Path != "body/1/row/0/cell/1/0" || chunks[3].Text != "B" {
		t.Fatalf("cell 1 nested paragraph chunk = %#v", chunks[3])
	}
}

func openDoc(t *testing.T, bodyXML string) *Document {
	t.Helper()

	data := testutil.BuildDocx(t, bodyXML, "")
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	return doc
}

func bulletNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="0">
    <w:lvl w:ilvl="0"><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/></w:lvl>
    <w:lvl w:ilvl="1"><w:numFmt w:val="bullet"/><w:lvlText w:val="•"/></w:lvl>
  </w:abstractNum>
  <w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`
}
