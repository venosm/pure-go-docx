package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"path"
	"strings"
	"time"
)

// Relationship type URIs used by the generated fixtures.
const (
	relTypeImage     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	relTypeHyperlink = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink"
	relTypeNumbering = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering"
	relTypeHeader    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/header"
	relTypeFooter    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer"
	relTypeFootnotes = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/footnotes"
	relTypeEndnotes  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/endnotes"
)

// Content types of the WordprocessingML parts used by the generated fixtures.
const (
	contentTypeDocument  = "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"
	contentTypeNumbering = "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"
	contentTypeHeader    = "application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"
	contentTypeFooter    = "application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"
	contentTypeFootnotes = "application/vnd.openxmlformats-officedocument.wordprocessingml.footnotes+xml"
	contentTypeEndnotes  = "application/vnd.openxmlformats-officedocument.wordprocessingml.endnotes+xml"
)

const xmlHeader = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`

// wordNamespaces are declared on every generated WordprocessingML part root.
const wordNamespaces = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" ` +
	`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" ` +
	`xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" ` +
	`xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" ` +
	`xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture" ` +
	`xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006" ` +
	`xmlns:wps="http://schemas.microsoft.com/office/word/2010/wordprocessingShape" ` +
	`xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml" ` +
	`mc:Ignorable="w14 wps"`

// fixedModTime keeps generated archives byte-stable across runs.
var fixedModTime = time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC)

// relationship is one entry of an OPC .rels part.
type relationship struct {
	id         string
	relType    string
	target     string
	targetMode string
}

// part is one stored archive entry.
type part struct {
	name string
	data []byte
}

// archive collects OPC parts and serializes them into a DOCX zip package.
type archive struct {
	parts     []part
	overrides []override
}

type override struct {
	partName    string
	contentType string
}

// defaultContentTypes cover the extensions used by every fixture.
var defaultContentTypes = []struct {
	extension   string
	contentType string
}{
	{"rels", "application/vnd.openxmlformats-package.relationships+xml"},
	{"xml", "application/xml"},
	{"png", "image/png"},
	{"jpeg", "image/jpeg"},
	{"jpg", "image/jpeg"},
	{"gif", "image/gif"},
	{"svg", "image/svg+xml"},
}

// newArchive returns an archive that already contains the package relationships.
func newArchive() *archive {
	a := &archive{}
	a.addPart("_rels/.rels", []byte(xmlHeader+"\n"+
		`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`+
		`<Relationship Id="rIdRoot1" `+
		`Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" `+
		`Target="word/document.xml"/>`+
		`</Relationships>`))
	return a
}

// addPart stores a raw part without registering a content-type override.
func (a *archive) addPart(name string, data []byte) {
	a.parts = append(a.parts, part{name: name, data: data})
}

// addXMLPart stores a part and registers its content-type override.
func (a *archive) addXMLPart(name, contentType string, data []byte) {
	a.addPart(name, data)
	a.overrides = append(a.overrides, override{partName: "/" + name, contentType: contentType})
}

// addDocument stores the main document part built from the supplied body XML.
func (a *archive) addDocument(bodyXML string) {
	a.addXMLPart("word/document.xml", contentTypeDocument, wordPartXML("document", "<w:body>"+bodyXML+"</w:body>"))
}

// addRelationships stores the .rels part belonging to sourcePart.
func (a *archive) addRelationships(sourcePart string, rels []relationship) {
	if len(rels) == 0 {
		return
	}

	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString("\n")
	b.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for _, rel := range rels {
		b.WriteString(`<Relationship Id="` + rel.id + `" Type="` + rel.relType + `" Target="` + escapeXML(rel.target) + `"`)
		if rel.targetMode != "" {
			b.WriteString(` TargetMode="` + rel.targetMode + `"`)
		}
		b.WriteString("/>")
	}
	b.WriteString("</Relationships>")

	dir, base := path.Split(sourcePart)
	a.addPart(path.Join(dir, "_rels", base+".rels"), []byte(b.String()))
}

// bytes serializes the archive into a DOCX package.
func (a *archive) bytes() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	all := append([]part{{name: "[Content_Types].xml", data: a.contentTypesXML()}}, a.parts...)
	for _, p := range all {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     p.name,
			Method:   zip.Deflate,
			Modified: fixedModTime,
		})
		if err != nil {
			return nil, fmt.Errorf("creating part %q: %w", p.name, err)
		}
		if _, err := w.Write(p.data); err != nil {
			return nil, fmt.Errorf("writing part %q: %w", p.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("closing archive: %w", err)
	}
	return buf.Bytes(), nil
}

func (a *archive) contentTypesXML() []byte {
	var b strings.Builder
	b.WriteString(xmlHeader)
	b.WriteString("\n")
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	for _, def := range defaultContentTypes {
		b.WriteString(`<Default Extension="` + def.extension + `" ContentType="` + def.contentType + `"/>`)
	}
	for _, o := range a.overrides {
		b.WriteString(`<Override PartName="` + o.partName + `" ContentType="` + o.contentType + `"/>`)
	}
	b.WriteString("</Types>")
	return []byte(b.String())
}

// wordPartXML wraps inner XML in a WordprocessingML root element.
func wordPartXML(rootLocal, innerXML string) []byte {
	return []byte(xmlHeader + "\n<w:" + rootLocal + " " + wordNamespaces + ">" + innerXML + "</w:" + rootLocal + ">")
}

// escapeXML escapes text for use in element content and attribute values.
func escapeXML(value string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(value)); err != nil {
		return value
	}
	return b.String()
}
