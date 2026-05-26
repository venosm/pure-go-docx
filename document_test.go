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
