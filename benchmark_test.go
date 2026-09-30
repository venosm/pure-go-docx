package godocx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/venosm/pure-go-docx/internal/testutil"
)

// benchmarkFixtures include generated and real-world test documents.
var benchmarkFixtures = []struct {
	name string
	file string
}{
	{name: "service-agreement", file: "service-agreement-en.docx"},
	{name: "service-contract-real", file: "service-contract-sample.docx"},
}

// scalingSizes define the number of blocks in a synthetic document.
var scalingSizes = []int{100, 1000, 10000}

// BenchmarkOpenReader measures parsing an archive in memory without disk I/O.
func BenchmarkOpenReader(b *testing.B) {
	for _, fx := range benchmarkFixtures {
		b.Run(fx.name, func(b *testing.B) {
			data := readFixture(b, fx.file)
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()

			for b.Loop() {
				if _, err := OpenReader(bytes.NewReader(data), int64(len(data))); err != nil {
					b.Fatalf("OpenReader() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkOpen measures opening a file from disk, including reading and parsing.
func BenchmarkOpen(b *testing.B) {
	for _, fx := range benchmarkFixtures {
		b.Run(fx.name, func(b *testing.B) {
			path := filepath.Join("testdata", fx.file)
			info, err := os.Stat(path)
			if err != nil {
				b.Fatalf("Stat() error = %v", err)
			}
			b.SetBytes(info.Size())
			b.ReportAllocs()

			for b.Loop() {
				if _, err := Open(path); err != nil {
					b.Fatalf("Open() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkToText measures rendering a parsed document as plain text.
func BenchmarkToText(b *testing.B) {
	for _, fx := range benchmarkFixtures {
		b.Run(fx.name, func(b *testing.B) {
			doc := openFixture(b, fx.file)
			b.ReportAllocs()

			for b.Loop() {
				_ = doc.ToText()
			}
		})
	}
}

// BenchmarkToMarkdown measures rendering a parsed document as Markdown.
func BenchmarkToMarkdown(b *testing.B) {
	for _, fx := range benchmarkFixtures {
		b.Run(fx.name, func(b *testing.B) {
			doc := openFixture(b, fx.file)
			b.ReportAllocs()

			for b.Loop() {
				_ = doc.ToMarkdown()
			}
		})
	}
}

// BenchmarkChunks measures splitting a parsed document into RAG chunks.
func BenchmarkChunks(b *testing.B) {
	for _, fx := range benchmarkFixtures {
		b.Run(fx.name, func(b *testing.B) {
			doc := openFixture(b, fx.file)
			b.ReportAllocs()

			for b.Loop() {
				_ = doc.Chunks()
			}
		})
	}
}

// BenchmarkAllImages measures lazy loading and decompression of all document images.
func BenchmarkAllImages(b *testing.B) {
	for _, file := range []string{"images.docx", "service-agreement-en.docx", "service-contract-sample.docx"} {
		b.Run(strings.TrimSuffix(file, ".docx"), func(b *testing.B) {
			doc := openFixture(b, file)
			b.ReportAllocs()

			for b.Loop() {
				if _, err := doc.AllImages(); err != nil {
					b.Fatalf("AllImages() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkOpenReaderScaling measures how parsing time grows with block count.
// Linear complexity keeps throughput in MB/s consistent across sizes.
func BenchmarkOpenReaderScaling(b *testing.B) {
	for _, blocks := range scalingSizes {
		b.Run(fmt.Sprintf("blocks=%d", blocks), func(b *testing.B) {
			data := testutil.BuildDocx(b, syntheticBody(blocks), "")
			b.SetBytes(int64(len(data)))
			b.ReportAllocs()

			for b.Loop() {
				if _, err := OpenReader(bytes.NewReader(data), int64(len(data))); err != nil {
					b.Fatalf("OpenReader() error = %v", err)
				}
			}
		})
	}
}

// BenchmarkChunksScaling measures how chunk generation time grows with block count.
func BenchmarkChunksScaling(b *testing.B) {
	for _, blocks := range scalingSizes {
		b.Run(fmt.Sprintf("blocks=%d", blocks), func(b *testing.B) {
			data := testutil.BuildDocx(b, syntheticBody(blocks), "")
			doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				b.Fatalf("OpenReader() error = %v", err)
			}
			b.ReportAllocs()

			for b.Loop() {
				_ = doc.Chunks()
			}
		})
	}
}

// syntheticBody creates a document body with the requested number of blocks.
// Every tenth block is a 3×3 table, and every fiftieth paragraph is a heading.
func syntheticBody(blocks int) string {
	var b strings.Builder
	for i := range blocks {
		switch {
		case i%10 == 9:
			b.WriteString(`<w:tbl>`)
			for row := range 3 {
				b.WriteString(`<w:tr>`)
				for column := range 3 {
					fmt.Fprintf(&b, `<w:tc><w:p><w:r><w:t>Cell %d-%d-%d</w:t></w:r></w:p></w:tc>`, i, row, column)
				}
				b.WriteString(`</w:tr>`)
			}
			b.WriteString(`</w:tbl>`)
		case i%50 == 0:
			fmt.Fprintf(&b, `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Chapter %d</w:t></w:r></w:p>`, i)
		default:
			fmt.Fprintf(&b, `<w:p><w:r><w:rPr><w:b/></w:rPr><w:t xml:space="preserve">Article %d </w:t></w:r>`+
				`<w:r><w:t>The supplier shall deliver the services in the scope and quality defined by this agreement.</w:t></w:r></w:p>`, i)
		}
	}
	return b.String()
}

func readFixture(tb testing.TB, file string) []byte {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		tb.Fatalf("ReadFile() error = %v", err)
	}
	return data
}

// openFixture parses a fixture in memory so lazy image loading remains available.
func openFixture(tb testing.TB, file string) *Document {
	tb.Helper()

	data := readFixture(tb, file)
	doc, err := OpenReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		tb.Fatalf("OpenReader() error = %v", err)
	}
	return doc
}
