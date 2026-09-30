package body

import (
	"context"
	"slices"
	"strings"
	"testing"
)

const (
	nsWord       = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	nsWordStrict = `xmlns:w="http://purl.oclc.org/ooxml/wordprocessingml/main"`
)

// TestParseDocumentOkrajoveXML zachycuje chování parseru na neobvyklém
// a poškozeném XML. Očekávané hodnoty odpovídají nestriktnímu režimu
// encoding/xml (Decoder.Strict = false), ze kterého parser vychází.
func TestParseDocumentOkrajoveXML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		xml       string
		wantTexty []string
		wantChyba string
	}{
		{
			name:      "prefix w",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a</w:t></w:r></w:p></w:body></w:document>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "vychozi jmenny prostor",
			xml:       `<document xmlns="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><body><p><r><t>a</t></r></p></body></document>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "jiny prefix",
			xml:       `<x:document xmlns:x="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><x:body><x:p><x:r><x:t>a</x:t></x:r></x:p></x:body></x:document>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "bez jmenneho prostoru",
			xml:       `<document><body><p><r><t>a</t></r></p></body></document>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "jmenny prostor deklarovany na predkovi",
			xml:       `<obal ` + nsWord + `><w:document><w:body><w:p><w:r><w:t>a</w:t></w:r></w:p></w:body></w:document></obal>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "strict OOXML",
			xml:       `<w:document ` + nsWordStrict + `><w:body></w:body></w:document>`,
			wantChyba: `unexpected document namespace "http://purl.oclc.org/ooxml/wordprocessingml/main"`,
		},
		{
			name:      "nedeklarovany prefix",
			xml:       `<w:document><w:body></w:body></w:document>`,
			wantChyba: `unexpected document namespace "w"`,
		},
		{
			name:      "prefix xml",
			xml:       `<xml:document><xml:body></xml:body></xml:document>`,
			wantChyba: `unexpected document namespace "http://www.w3.org/XML/1998/namespace"`,
		},
		{
			name:      "useknuty uvnitr textu",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a`,
			wantChyba: "XML syntax error on line 1: unexpected EOF",
		},
		{
			name:      "useknuty po konci body",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a</w:t></w:r></w:p></w:body>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "neuzavreny beh",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a</w:t></w:p><w:p><w:r><w:t>b</w:t></w:r></w:p></w:body></w:document>`,
			wantTexty: []string{"a", "b"},
		},
		{
			name:      "neuzavreny text",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a</w:r></w:p></w:body></w:document>`,
			wantTexty: []string{"a"},
		},
		{
			// Přebytečný koncový tag uzavře w:body a zbytek těla se nečte.
			name:      "prebytecny koncovy tag",
			xml:       `<w:document ` + nsWord + `><w:body><w:p><w:r><w:t>a</w:t></w:r></w:p></w:tbl><w:p><w:r><w:t>b</w:t></w:r></w:p></w:body></w:document>`,
			wantTexty: []string{"a"},
		},
		{
			name:      "jiny prefix v koncovem tagu",
			xml:       `<w:document ` + nsWord + ` xmlns:x="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>a</w:t></w:r></x:p></w:body></w:document>`,
			wantChyba: "XML syntax error on line 1: element <p> in space w closed by </p> in space x",
		},
		{
			name:      "koncovy tag bez zacatku",
			xml:       `</w:document>`,
			wantChyba: "XML syntax error on line 1: unexpected end element </document>",
		},
		{
			name:      "prazdny vstup",
			xml:       ``,
			wantChyba: "document root not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bloky, err := ParseDocument(context.Background(), strings.NewReader(tc.xml), nil, nil, "word/document.xml")
			if tc.wantChyba != "" {
				if err == nil || err.Error() != tc.wantChyba {
					t.Fatalf("ParseDocument() error = %v, want %q", err, tc.wantChyba)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDocument() error = %v", err)
			}
			if got := textyOdstavcu(bloky); !slices.Equal(got, tc.wantTexty) {
				t.Fatalf("texty odstavců = %q, want %q", got, tc.wantTexty)
			}
		})
	}
}

func textyOdstavcu(bloky []Block) []string {
	var texty []string
	for _, blok := range bloky {
		odstavec, ok := blok.(*Paragraph)
		if !ok {
			continue
		}
		var b strings.Builder
		for _, run := range odstavec.Runs {
			b.WriteString(run.Text)
		}
		texty = append(texty, b.String())
	}
	return texty
}
