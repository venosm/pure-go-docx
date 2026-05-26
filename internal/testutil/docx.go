package testutil

import (
	"archive/zip"
	"bytes"
	"testing"
)

// File is an additional file stored in a synthetic DOCX archive.
type File struct {
	Name string
	Data []byte
}

// BuildDocx creates a minimal DOCX archive with the supplied body XML.
func BuildDocx(t testing.TB, bodyXML, relsXML string, files ...File) []byte {
	t.Helper()

	allFiles := []File{
		{Name: "[Content_Types].xml", Data: []byte(contentTypesXML())},
		{Name: "word/document.xml", Data: []byte(documentXML(bodyXML))},
	}
	if relsXML != "" {
		allFiles = append(allFiles, File{
			Name: "word/_rels/document.xml.rels",
			Data: []byte(relationshipsXML(relsXML)),
		})
	}
	allFiles = append(allFiles, files...)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, file := range allFiles {
		w, err := zw.Create(file.Name)
		if err != nil {
			t.Fatalf("Create(%q) error = %v", file.Name, err)
		}
		if _, err := w.Write(file.Data); err != nil {
			t.Fatalf("Write(%q) error = %v", file.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buf.Bytes()
}

func contentTypesXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="jpg" ContentType="image/jpeg"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
  <Default Extension="gif" ContentType="image/gif"/>
  <Default Extension="svg" ContentType="image/svg+xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`
}

func documentXML(bodyXML string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"
  xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
  xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
  xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
  xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">
  <w:body>` + bodyXML + `</w:body>
</w:document>`
}

func relationshipsXML(relsXML string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + relsXML + `</Relationships>`
}
