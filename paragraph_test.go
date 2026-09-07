package godocx

import (
	"bytes"
	"testing"

	"github.com/venosm/pure-go-docx/internal/testutil"
)

func TestParagraph_ListItem_Decimal(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>first</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>second</w:t></w:r></w:p>`, decimalNumberingXML())

	assertParagraphList(t, doc.Body[0], ListRef{
		NumID:     1,
		Level:     0,
		Format:    "decimal",
		LevelText: "%1.",
		Ordinal:   1,
	})
	assertParagraphList(t, doc.Body[1], ListRef{
		NumID:     1,
		Level:     0,
		Format:    "decimal",
		LevelText: "%1.",
		Ordinal:   2,
	})
}

func TestParagraph_ListItem_Nested(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>first</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>nested</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>second</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>nested again</w:t></w:r></w:p>`, nestedNumberingXML())

	assertParagraphList(t, doc.Body[0], ListRef{NumID: 1, Level: 0, Format: "decimal", LevelText: "%1.", Ordinal: 1})
	assertParagraphList(t, doc.Body[1], ListRef{NumID: 1, Level: 1, Format: "lowerLetter", LevelText: "%2.", Ordinal: 1})
	assertParagraphList(t, doc.Body[2], ListRef{NumID: 1, Level: 0, Format: "decimal", LevelText: "%1.", Ordinal: 2})
	assertParagraphList(t, doc.Body[3], ListRef{NumID: 1, Level: 1, Format: "lowerLetter", LevelText: "%2.", Ordinal: 1})
}

func TestParagraph_NoList_WhenNumIDZero(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="0"/></w:numPr></w:pPr><w:r><w:t>not list</w:t></w:r></w:p>`, decimalNumberingXML())

	paragraph := requireParagraph(t, doc.Body[0])
	if paragraph.List != nil {
		t.Fatalf("Paragraph.List = %#v, want nil", paragraph.List)
	}
}

func TestParagraph_NoList_WhenResolverEmpty(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>not list</w:t></w:r></w:p>`, "")
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	paragraph := requireParagraph(t, doc.Body[0])
	if paragraph.List != nil {
		t.Fatalf("Paragraph.List = %#v, want nil", paragraph.List)
	}
}

func TestParagraph_List_WithEmptyIlvl(t *testing.T) {
	t.Parallel()

	doc := openNumberedDoc(t, `
<w:p><w:pPr><w:numPr><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>first</w:t></w:r></w:p>`, decimalNumberingXML())

	assertParagraphList(t, doc.Body[0], ListRef{
		NumID:     1,
		Level:     0,
		Format:    "decimal",
		LevelText: "%1.",
		Ordinal:   1,
	})
}

func TestParagraph_AlternateContentChoice(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<mc:AlternateContent>
  <mc:Choice Requires="w14"><w:p><w:r><w:t>choice paragraph</w:t></w:r></w:p></mc:Choice>
  <mc:Fallback><w:p><w:r><w:t>fallback paragraph</w:t></w:r></w:p></mc:Fallback>
</mc:AlternateContent>
<mc:AlternateContent>
  <mc:Fallback><w:p><w:r><w:t>fallback only</w:t></w:r></w:p></mc:Fallback>
</mc:AlternateContent>
<w:p>
  <w:r>
    <mc:AlternateContent>
      <mc:Choice Requires="w14"><w:t>choice run</w:t></mc:Choice>
      <mc:Fallback><w:t>fallback run</w:t></mc:Fallback>
    </mc:AlternateContent>
  </w:r>
</w:p>`)

	if got, want := doc.ToText(), "choice paragraph\nfallback only\nchoice run\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestParagraph_FieldDisplayValueState(t *testing.T) {
	t.Parallel()

	doc := openDoc(t, `
<w:p>
  <w:r><w:t xml:space="preserve">Before </w:t></w:r>
  <w:r><w:fldChar w:fldCharType="begin"/></w:r>
  <w:r><w:instrText xml:space="preserve"> HYPERLINK "https://example.com" </w:instrText></w:r>
  <w:r><w:t>hidden</w:t></w:r>
  <w:r><w:fldChar w:fldCharType="separate"/></w:r>
  <w:r><w:t>Example</w:t></w:r>
  <w:r><w:fldChar w:fldCharType="end"/></w:r>
  <w:r><w:t xml:space="preserve"> </w:t></w:r>
  <w:fldSimple w:instr='HYPERLINK "https://simple.example"'><w:r><w:t>Simple</w:t></w:r></w:fldSimple>
</w:p>`)

	if got, want := doc.ToText(), "Before Example Simple\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
	if got, want := doc.ToMarkdown(), "Before [Example](https://example.com) [Simple](https://simple.example)\n"; got != want {
		t.Fatalf("ToMarkdown() = %q, want %q", got, want)
	}
}

func openNumberedDoc(t *testing.T, bodyXML, numberingXML string) *Document {
	t.Helper()

	data := testutil.BuildDocx(t, bodyXML, `
<Relationship Id="rIdNumbering" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>`,
		testutil.File{Name: "word/numbering.xml", Data: []byte(numberingXML)},
	)
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}
	return doc
}

func assertParagraphList(t *testing.T, block Block, want ListRef) {
	t.Helper()

	paragraph := requireParagraph(t, block)
	if paragraph.List == nil {
		t.Fatal("Paragraph.List = nil, want list ref")
	}
	if *paragraph.List != want {
		t.Fatalf("Paragraph.List = %#v, want %#v", *paragraph.List, want)
	}
}

func requireParagraph(t *testing.T, block Block) *Paragraph {
	t.Helper()

	paragraph, ok := block.(*Paragraph)
	if !ok {
		t.Fatalf("block type = %T, want *Paragraph", block)
	}
	return paragraph
}

func decimalNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="0">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="decimal"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`
}

func nestedNumberingXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:abstractNum w:abstractNumId="0">
    <w:lvl w:ilvl="0">
      <w:start w:val="1"/>
      <w:numFmt w:val="decimal"/>
      <w:lvlText w:val="%1."/>
    </w:lvl>
    <w:lvl w:ilvl="1">
      <w:start w:val="1"/>
      <w:numFmt w:val="lowerLetter"/>
      <w:lvlText w:val="%2."/>
    </w:lvl>
  </w:abstractNum>
  <w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>
</w:numbering>`
}
