package godocx

import (
	"path/filepath"
	"testing"
)

func TestDocxFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filename      string
		minBodyBlocks int
		minTextLen    int
		minChunks     int
		minImages     int
		headers       int
		footers       int
		footnotes     int
		endnotes      int
	}{
		{
			name:          "service agreement",
			filename:      "service-agreement-en.docx",
			minBodyBlocks: 20,
			minTextLen:    1000,
			minChunks:     20,
			headers:       2,
			footers:       1,
			footnotes:     3,
			endnotes:      0,
		},
		{
			name:          "real service contract",
			filename:      "service-contract-sample.docx",
			minBodyBlocks: 100,
			minTextLen:    40000,
			minChunks:     200,
			minImages:     1,
			headers:       2,
			footers:       2,
			footnotes:     0,
			endnotes:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := Open(filepath.Join("testdata", tc.filename))
			if err != nil {
				t.Fatalf("Open() error = %v", err)
			}

			text := doc.ToText()
			markdown := doc.ToMarkdown()
			chunks := doc.Chunks()
			t.Logf(
				"body=%d headers=%d footers=%d footnotes=%d endnotes=%d text=%d markdown=%d chunks=%d",
				len(doc.Body),
				len(doc.Headers),
				len(doc.Footers),
				len(doc.Footnotes),
				len(doc.Endnotes),
				len(text),
				len(markdown),
				len(chunks),
			)

			if len(doc.Body) < tc.minBodyBlocks {
				t.Fatalf("len(Body) = %d, want at least %d", len(doc.Body), tc.minBodyBlocks)
			}
			if len(text) < tc.minTextLen {
				t.Fatalf("len(ToText()) = %d, want at least %d", len(text), tc.minTextLen)
			}
			if markdown == "" {
				t.Fatal("ToMarkdown() is empty")
			}
			if len(chunks) < tc.minChunks {
				t.Fatalf("len(Chunks()) = %d, want at least %d", len(chunks), tc.minChunks)
			}
			if tc.minImages > 0 {
				images, err := doc.AllImages()
				if err != nil {
					t.Fatalf("AllImages() error = %v", err)
				}
				if len(images) < tc.minImages {
					t.Fatalf("len(AllImages()) = %d, want at least %d", len(images), tc.minImages)
				}
			}
			if len(doc.Headers) != tc.headers {
				t.Fatalf("len(Headers) = %d, want %d", len(doc.Headers), tc.headers)
			}
			if len(doc.Footers) != tc.footers {
				t.Fatalf("len(Footers) = %d, want %d", len(doc.Footers), tc.footers)
			}
			if len(doc.Footnotes) != tc.footnotes {
				t.Fatalf("len(Footnotes) = %d, want %d", len(doc.Footnotes), tc.footnotes)
			}
			if len(doc.Endnotes) != tc.endnotes {
				t.Fatalf("len(Endnotes) = %d, want %d", len(doc.Endnotes), tc.endnotes)
			}
		})
	}
}
