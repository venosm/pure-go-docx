# pure-go-docx

[![Go Reference](https://pkg.go.dev/badge/github.com/venosm/pure-go-docx.svg)](https://pkg.go.dev/github.com/venosm/pure-go-docx)
[![Go Report Card](https://goreportcard.com/badge/github.com/venosm/pure-go-docx)](https://goreportcard.com/report/github.com/venosm/pure-go-docx)

High-performance, pure-Go library for parsing and extracting structural content from Microsoft Word (`.docx` / OOXML) documents.

Built specifically for **LLM / RAG ingestion pipelines**, search indexing, procurement analysis, and document processing systems where visual rendering is unnecessary, but semantic structure, speed, and memory predictability are critical.

---

## Highlights

- **Pure Go & Zero External Dependencies**: Relies exclusively on the Go standard library (`archive/zip`, `encoding/xml`). No Cgo, no LibreOffice, and no external binary dependencies.
- **High Performance & Streaming XML**: Employs `encoding/xml.Decoder.RawToken` with custom element tracking to bypass namespace URI resolution overhead. Measured parsing throughput is ~9–13 MB/s on the benchmark fixtures.
- **OOXML Structural Coverage**:
  - **Body & Related Parts**: Document body, headers, footers, footnotes, and endnotes.
  - **Formatted Text**: Headings (levels 1–9), runs with bold, italic, underline, tabs, breaks, and hyperlinks.
  - **Lists & Ordinals**: Multi-level numbered and bulleted lists with resolved ordinals (e.g. `1.`, `1.1`, `a.`) via relationship-driven `numbering.xml` parsing.
  - **Advanced Tables**: Dense 2D grid construction, horizontal merge (`gridSpan`) expansion, and vertical merge (`vMerge`) continuation inheritance.
  - **Word Markup Quirks**: Gracefully resolves `mc:AlternateContent` (Choice vs. Fallback), complex fields (`w:instrText` vs. display runs), simple fields (`w:fldSimple`), and structured document tags (SDT content controls).
  - **Lazy Media Extraction**: Referenced images remain in the DOCX archive and are decompressed on demand.
- **Multi-Format Extraction**:
  - `Document.ToText()`: Fast, clean plain-text linearization.
  - `Document.ToMarkdown()`: Rich Markdown preserving headers, tables, links, images, and notes.
  - `Document.Chunks()`: Pre-segmented, production-ready RAG chunks with hierarchical addressing paths, source references, and table/image/list metadata.

---

## Performance & Benchmarks

The library is optimized to minimize heap allocations. The benchmarks below measure parsing and chunking on representative fixtures and synthetic documents.

Measured on an **Intel® Core™ i7-1165G7 @ 2.80GHz** (`linux/amd64`):

| Operation | Target / Fixture | Time | Throughput | Memory / allocations |
| :--- | :--- | :--- | :--- | :--- |
| **`OpenReader`** | Generated service agreement (clean) | **648 µs** | **12.77 MB/s** | 267 KB (5 384 allocs) |
| **`OpenReader`** | Real-world service contract (Word-authored) | **5.87 ms** | **9.37 MB/s** | 2.34 MB (51 523 allocs) |
| **`Open`** (disk) | Real-world service contract (Word-authored) | **5.82 ms** | **9.44 MB/s** | 2.47 MB (51 544 allocs) |
| **`ToText`** | Real-world service contract | **87.1 µs** | — | 307 KB (430 allocs) |
| **`ToMarkdown`** | Real-world service contract | **144.2 µs** | — | 373 KB (438 allocs) |
| **`Chunks`** (RAG) | Real-world service contract | **264.0 µs** | — | 386 KB (2 130 allocs) |
| **`AllImages`** | Embedded image decompression (lazy) | **2–15 µs** | — | 2–24 KB (19–44 allocs) |

### Measured Scaling

Scaling benchmarks on synthetic documents with mixed paragraphs, headings, and 3×3 tables show approximately linear growth for these inputs:

- **100 blocks**: Parsed in **0.63 ms** (313 KB memory)
- **1,000 blocks**: Parsed in **5.89 ms** (2.88 MB memory, **9.35×** time)
- **10,000 blocks** (~book-length): Parsed in **56.9 ms** (28.7 MB memory, **9.67×** time) and chunked in **8.68 ms** (9.47 MB memory)

> For full benchmark details and test methodology, see [bench.md](bench.md).

---

## Installation

```bash
go get github.com/venosm/pure-go-docx
```

Requires Go 1.26 or later.

---

## Quick Start

### 1. Plain Text Extraction

```go
package main

import (
	"fmt"
	"log"

	godocx "github.com/venosm/pure-go-docx"
)

func main() {
	doc, err := godocx.Open("contract.docx")
	if err != nil {
		log.Fatalf("failed to open docx: %v", err)
	}

	// Linearized text representation (tabs for tables, resolved list markers)
	fmt.Println(doc.ToText())
}
```

### 2. RAG Chunking Pipeline

`doc.Chunks()` returns pre-segmented, contextual chunks ideal for vector database indexing or embedding generation:

```go
doc, err := godocx.Open("contract.docx")
if err != nil {
	log.Fatal(err)
}

for _, chunk := range doc.Chunks() {
	fmt.Printf("[%s] %s\n", chunk.Path, chunk.Source)
	fmt.Println(chunk.Markdown)

	if chunk.Table != nil {
		fmt.Printf("  -> Table row %d with %d columns\n", chunk.Table.Row, chunk.Table.Columns)
	}
	if chunk.List != nil {
		fmt.Printf("  -> List item (level %d, marker: %q)\n", chunk.List.Level, chunk.List.Marker)
	}
	if chunk.Image != nil {
		fmt.Printf("  -> Image reference: %s\n", chunk.Image.Filename)
	}
	fmt.Println("---")
}
```

### 3. Markdown Export

Export the complete document—including headers, footers, tables, links, and footnotes—as clean Markdown:

```go
doc, err := godocx.Open("contract.docx")
if err != nil {
	log.Fatal(err)
}

markdown := doc.ToMarkdown()
fmt.Println(markdown)
```

### 4. Lazy Image Extraction

Embedded images are not loaded during initial parsing. Instead, image bytes are read and decompressed on demand directly from the archive:

```go
doc, err := godocx.Open("contract.docx")
if err != nil {
	log.Fatal(err)
}

// Extract a specific image by relationship ID:
img, err := doc.Image("rId5")
if err != nil {
	log.Fatalf("image not found: %v", err)
}
fmt.Printf("File: %s, MIME: %s, Size: %d bytes\n", img.Filename, img.ContentType, len(img.Bytes))

// Or extract all referenced images across body and related parts:
allImages, err := doc.AllImages()
if err != nil {
	log.Fatal(err)
}
for _, image := range allImages {
	fmt.Printf("Image ID %s: %s (%d bytes)\n", image.ID, image.Filename, len(image.Bytes))
}
```

### 5. In-Memory / Random-Access Reader Loading

To parse DOCX files from S3, memory buffers, or custom storage without saving to disk:

```go
var data []byte // e.g. fetched over HTTP or read from object storage
reader := bytes.NewReader(data)

doc, err := godocx.OpenReader(reader, int64(len(data)))
if err != nil {
	log.Fatal(err)
}

// Note: The reader must remain valid for the lifetime of doc if lazy images are read.
```

---

## Architectural Details

### Memory Management & Lifetime

- **`godocx.Open(path)`**: Reads the entire archive into a memory buffer (up to 500 MiB by default). The underlying OS file descriptor is closed immediately before `Open` returns. This guarantees that `doc.Image()` and `doc.AllImages()` remain completely functional for the lifetime of the `Document` object without leaking open file handles.
- **`godocx.OpenReader(r, size)`**: Uses the caller's `io.ReaderAt` directly without allocating a duplicate buffer. Callers must keep the `ReaderAt` valid as long as lazy image loading is used.

### RAG-First Table Merge Resolution

Standard DOCX tables often span cells across rows (`w:vMerge`) or columns (`w:gridSpan`). In naïve text extractors, merged rows lose their header or context when split into search chunks.

`pure-go-docx` resolves tables into a dense 2D grid:
- **`gridSpan` (Horizontal Merge)**: Expanded so the origin cell contains the content, while covered horizontal columns are present but empty, avoiding duplicated tokens.
- **`vMerge` (Vertical Continuation)**: Continuation cells inherit the `Blocks` and text of the origin cell, retaining merged-cell context in table-row chunks.

### Streaming XML Tokenization (`RawToken`)

Standard Go `xml.Decoder.Token()` allocates and resolves XML namespace URIs on every single token. Because WordprocessingML documents are namespace-heavy, `pure-go-docx` uses `Decoder.RawToken()` combined with an internal tag-matching stack:
- Element names are evaluated by local tags.
- Non-strict element closure matches Word's fault-tolerant markup behavior.
- Namespace URI translation is evaluated only when inspecting root elements (`w:document`, `w:hdr`, `w:ftr`, etc.).

---

## Testing & Fixtures

The test suite balances synthetic unit tests with real-world document validation:

- **Synthetic In-Memory Tests**: Unit tests synthesize byte-precise DOCX archives in memory using helper builders to test specific edge cases (deeply nested lists, broken XML, malformed namespaces, complex fields).
- **English Feature Fixtures**: Generated by `testdata/gen` to test each parser feature area independently (`tables.docx`, `lists-nested.docx`, `headers-footers.docx`, etc.).
- **Real-World Fixture**: Validated against real procurement documents (`testdata/service-contract-sample.docx`) with thousands of words, complex styles, and tables.

---

## Verification & Development

To run the verification suite:

```bash
make tidy        # Check and tidy dependencies
make test        # Run unit tests across all packages
make bench       # Run benchmark suite with memory reporting
make lint        # Run go vet
make build       # Build project
```

Race detection can be verified with:

```bash
go test ./... -race
```

---

## Scope & Limitations

`pure-go-docx` is focused strictly on fast, read-only extraction and structural ingestion:
- **Read-Only**: Creating or modifying `.docx` files is out of scope.
- **OOXML Format Only**: Supports Microsoft Word `.docx` files (ECMA-376 / ISO/IEC 29500). Legacy binary `.doc` files (Word 97–2003) are not supported.
- **Layout Engine Free**: Does not compute visual page layout, font metrics, line wraps, or visual pagination.
