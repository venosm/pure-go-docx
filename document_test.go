package godocx

import (
	"bytes"
	"testing"

	"github.com/venosm/pure-go-docx/internal/testutil"
)

func TestOpenReaderParagraphsAndText(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:p>
  <w:pPr><w:pStyle w:val="Heading1"/></w:pPr>
  <w:r><w:t>Nadpis</w:t></w:r>
</w:p>
<w:p>
  <w:r><w:rPr><w:b/><w:i/></w:rPr><w:t xml:space="preserve">slovo </w:t></w:r>
  <w:r><w:t>stěna</w:t><w:tab/><w:t>řádek</w:t><w:br/><w:t>nový</w:t></w:r>
</w:p>`, "")

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	if len(doc.Body) != 2 {
		t.Fatalf("len(Body) = %d, want 2", len(doc.Body))
	}
	heading, ok := doc.Body[0].(*Paragraph)
	if !ok {
		t.Fatalf("Body[0] type = %T, want *Paragraph", doc.Body[0])
	}
	if heading.StyleID != "Heading1" || heading.HeadingLvl != 1 {
		t.Fatalf("heading style = %q level %d, want Heading1 level 1", heading.StyleID, heading.HeadingLvl)
	}
	if got, want := doc.ToText(), "Nadpis\nslovo stěna\třádek\nnový\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestOpenReaderTables(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:tbl>
  <w:tr>
    <w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc>
  </w:tr>
  <w:tr>
    <w:tc><w:p><w:r><w:t>C</w:t></w:r></w:p></w:tc>
    <w:tc><w:p><w:r><w:t>D</w:t></w:r></w:p></w:tc>
  </w:tr>
</w:tbl>`, "")

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	table, ok := doc.Body[0].(*Table)
	if !ok {
		t.Fatalf("Body[0] type = %T, want *Table", doc.Body[0])
	}
	if got, want := len(table.Grid), 2; got != want {
		t.Fatalf("rows = %d, want %d", got, want)
	}
	if got, want := doc.ToText(), "A\tB\nC\tD\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestOpenReaderImages(t *testing.T) {
	t.Parallel()

	imageBytes := []byte{0x89, 'P', 'N', 'G'}
	data := testutil.BuildDocx(t, `
<w:p>
  <w:r>
    <w:drawing>
      <wp:inline>
        <wp:extent cx="914400" cy="914400"/>
        <wp:docPr id="1" name="image" descr="schema zakazky"/>
        <a:graphic>
          <a:graphicData>
            <pic:pic>
              <pic:blipFill><a:blip r:embed="rId5"/></pic:blipFill>
            </pic:pic>
          </a:graphicData>
        </a:graphic>
      </wp:inline>
    </w:drawing>
  </w:r>
</w:p>`, `
<Relationship Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/>`,
		testutil.File{Name: "word/media/image1.png", Data: imageBytes},
	)

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	if got, want := doc.ToText(), "[image: word/media/image1.png]\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
	image, err := doc.Image("rId5")
	if err != nil {
		t.Fatalf("Image() error = %v", err)
	}
	if !bytes.Equal(image.Bytes, imageBytes) {
		t.Fatalf("Image().Bytes = %v, want %v", image.Bytes, imageBytes)
	}
	if image.ContentType != "image/png" {
		t.Fatalf("Image().ContentType = %q, want image/png", image.ContentType)
	}
	if image.AltText != "schema zakazky" {
		t.Fatalf("Image().AltText = %q, want schema zakazky", image.AltText)
	}

	images, err := doc.AllImages()
	if err != nil {
		t.Fatalf("AllImages() error = %v", err)
	}
	if len(images) != 1 || images[0].RelID != "rId5" {
		t.Fatalf("AllImages() = %#v, want one rId5 image", images)
	}
}

func TestOpenReaderRelatedPartsAndNotes(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:p>
  <w:r><w:t>Body</w:t></w:r>
  <w:r><w:footnoteReference w:id="2"/></w:r>
  <w:r><w:t xml:space="preserve"> end</w:t></w:r>
  <w:r><w:endnoteReference w:id="3"/></w:r>
</w:p>`, `
<Relationship Id="rIdHeader" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
<Relationship Id="rIdFooter" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>
<Relationship Id="rIdFootnotes" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes" Target="footnotes.xml"/>
<Relationship Id="rIdEndnotes" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/endnotes" Target="endnotes.xml"/>`,
		testutil.File{Name: "word/header1.xml", Data: testutil.WordPartXML("hdr", `<w:p><w:r><w:t>Header</w:t></w:r></w:p>`)},
		testutil.File{Name: "word/footer1.xml", Data: testutil.WordPartXML("ftr", `<w:p><w:r><w:t>Footer</w:t></w:r></w:p>`)},
		testutil.File{Name: "word/footnotes.xml", Data: testutil.WordPartXML("footnotes", `
<w:footnote w:id="-1" w:type="separator"><w:p><w:r><w:t>skip</w:t></w:r></w:p></w:footnote>
<w:footnote w:id="2"><w:p><w:r><w:t>Footnote text</w:t></w:r></w:p></w:footnote>`)},
		testutil.File{Name: "word/endnotes.xml", Data: testutil.WordPartXML("endnotes", `<w:endnote w:id="3"><w:p><w:r><w:t>Endnote text</w:t></w:r></w:p></w:endnote>`)},
	)

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	if got, want := paragraphText(requireParagraph(t, doc.Headers["rIdHeader"][0])), "Header"; got != want {
		t.Fatalf("header text = %q, want %q", got, want)
	}
	if got, want := paragraphText(requireParagraph(t, doc.Footers["rIdFooter"][0])), "Footer"; got != want {
		t.Fatalf("footer text = %q, want %q", got, want)
	}
	if _, ok := doc.Footnotes["-1"]; ok {
		t.Fatal("separator footnote was parsed, want skipped")
	}
	if got, want := paragraphText(requireParagraph(t, doc.Footnotes["2"][0])), "Footnote text"; got != want {
		t.Fatalf("footnote text = %q, want %q", got, want)
	}
	if got, want := paragraphText(requireParagraph(t, doc.Endnotes["3"][0])), "Endnote text"; got != want {
		t.Fatalf("endnote text = %q, want %q", got, want)
	}
	if got, want := doc.ToText(), "Body[footnote: 2] end[endnote: 3]\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
}

func TestOpenReaderRelatedPartImages(t *testing.T) {
	t.Parallel()

	imageBytes := []byte{0x89, 'P', 'N', 'G', 1}
	data := testutil.BuildDocx(t, `<w:p><w:r><w:t>Body</w:t></w:r></w:p>`, `
<Relationship Id="rIdHeader" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>`,
		testutil.File{Name: "word/header1.xml", Data: testutil.WordPartXML("hdr", `
<w:p>
  <w:r>
    <w:drawing>
      <wp:inline>
        <wp:extent cx="100" cy="200"/>
        <wp:docPr id="1" name="header-image" descr="header schema"/>
        <a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rIdImage"/></pic:blipFill></pic:pic></a:graphicData></a:graphic>
      </wp:inline>
    </w:drawing>
  </w:r>
</w:p>`)},
		testutil.File{Name: "word/_rels/header1.xml.rels", Data: []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rIdImage" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/header.png"/>
</Relationships>`)},
		testutil.File{Name: "word/media/header.png", Data: imageBytes},
	)

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	images, err := doc.AllImages()
	if err != nil {
		t.Fatalf("AllImages() error = %v", err)
	}
	if len(images) != 1 {
		t.Fatalf("len(AllImages()) = %d, want 1", len(images))
	}
	if got, want := images[0].ID, "word/header1.xml#rIdImage"; got != want {
		t.Fatalf("image ID = %q, want %q", got, want)
	}
	if got, want := images[0].PartName, "word/header1.xml"; got != want {
		t.Fatalf("image part = %q, want %q", got, want)
	}
	if !bytes.Equal(images[0].Bytes, imageBytes) {
		t.Fatalf("image bytes = %v, want %v", images[0].Bytes, imageBytes)
	}

	image, err := doc.Image("word/header1.xml#rIdImage")
	if err != nil {
		t.Fatalf("Image(header ID) error = %v", err)
	}
	if image.AltText != "header schema" {
		t.Fatalf("Image().AltText = %q, want header schema", image.AltText)
	}
}

func TestOpenReaderAlternateContentDrawingChoosesChoice(t *testing.T) {
	t.Parallel()

	data := testutil.BuildDocx(t, `
<w:p>
  <w:r>
    <w:drawing>
      <mc:AlternateContent>
        <mc:Choice Requires="w14">
          <wp:inline>
            <wp:docPr id="1" name="choice" descr="choice image"/>
            <a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rIdChoice"/></pic:blipFill></pic:pic></a:graphicData></a:graphic>
          </wp:inline>
        </mc:Choice>
        <mc:Fallback>
          <wp:inline>
            <wp:docPr id="2" name="fallback" descr="fallback image"/>
            <a:graphic><a:graphicData><pic:pic><pic:blipFill><a:blip r:embed="rIdFallback"/></pic:blipFill></pic:pic></a:graphicData></a:graphic>
          </wp:inline>
        </mc:Fallback>
      </mc:AlternateContent>
    </w:drawing>
  </w:r>
</w:p>`, `
<Relationship Id="rIdChoice" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/choice.png"/>
<Relationship Id="rIdFallback" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/fallback.png"/>`,
		testutil.File{Name: "word/media/choice.png", Data: []byte{1}},
		testutil.File{Name: "word/media/fallback.png", Data: []byte{2}},
	)

	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	if got, want := doc.ToText(), "[image: word/media/choice.png]\n"; got != want {
		t.Fatalf("ToText() = %q, want %q", got, want)
	}
	images, err := doc.AllImages()
	if err != nil {
		t.Fatalf("AllImages() error = %v", err)
	}
	if len(images) != 1 || images[0].RelID != "rIdChoice" || images[0].AltText != "choice image" {
		t.Fatalf("AllImages() = %#v, want choice image only", images)
	}
}
