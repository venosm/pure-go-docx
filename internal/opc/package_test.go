package opc

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestPackageMainDocumentPartUsesContentTypes(t *testing.T) {
	t.Parallel()

	data := makeZip(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/custom/main.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"custom/main.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
	})

	pkg, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	part, err := pkg.MainDocumentPart()
	if err != nil {
		t.Fatalf("MainDocumentPart() error = %v", err)
	}
	if part != "custom/main.xml" {
		t.Fatalf("MainDocumentPart() = %q, want %q", part, "custom/main.xml")
	}
}

func TestRelationshipsResolveRelativeTargets(t *testing.T) {
	t.Parallel()

	data := makeZip(t, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`,
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
		"word/_rels/document.xml.rels": `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/image1.png"/>
</Relationships>`,
		"word/media/image1.png": "image",
	})

	pkg, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("OpenReader() error = %v", err)
	}

	rels, err := pkg.Relationships("word/document.xml")
	if err != nil {
		t.Fatalf("Relationships() error = %v", err)
	}
	if got, want := rels["rId1"].Target, "word/media/image1.png"; got != want {
		t.Fatalf("resolved target = %q, want %q", got, want)
	}
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("Write(%q) error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buf.Bytes()
}
