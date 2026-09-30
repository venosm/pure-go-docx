# pure-go-docx

Pure-Go DOCX reader library for extracting text, lists, tables, and embedded
images from Microsoft Word OOXML documents.

The library is intended for document ingestion pipelines where rendering is not
needed, but structural text output matters: RAG indexing, search extraction,
procurement document processing, and similar backend workflows.

## Module

```text
github.com/venosm/pure-go-docx
```

## Status

Milestones 1 and 2 are implemented and committed.

Implemented:

- OPC zip loading with content-type based main document discovery.
- Relationship parsing for document media and numbering definitions.
- Streaming XML parsing with `encoding/xml`. Body, header, footer, and note
  parts are read with `Decoder.RawToken` and the parser checks element nesting
  itself, matching `Decoder.Token` in non-strict mode.
- Paragraphs, runs, tabs, line breaks, headings, and basic run formatting flags.
- Numbered and bulleted list resolution with nesting and ordinals.
- Flat and nested tables.
- Table `gridSpan` expansion into dense grids.
- Table `vMerge` continuation inheritance so each row is self-contained.
- Headers and footers parsed from related WordprocessingML parts.
- Footnotes and endnotes parsed by note ID, with in-body references.
- `mc:AlternateContent` choice/fallback handling for blocks, runs, and drawings.
- Field display-value state handling for complex fields and `w:fldSimple`.
- Embedded image references and lazy image byte loading.
- Part-aware image IDs for related parts, while body image `rId` lookup remains supported.
- Plain-text output through `Document.ToText()`.
- Full Markdown output through `Document.ToMarkdown()` for RAG ingestion.
- RAG chunks with source, path, Markdown, list, table, image, and note metadata.
- Synthetic DOCX unit tests built in memory.

Not implemented yet:

- DOCX writing.
- Legacy binary `.doc` support.

## Usage

```go
package main

import (
	"fmt"
	"log"

	godocx "github.com/venosm/pure-go-docx"
)

func main() {
	doc, err := godocx.Open("document.docx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(doc.ToText())
}
```

`Open` reads the whole file into memory (up to 500 MiB), so the returned
`Document` can still load images lazily after the file is closed.

Open from any random-access reader:

```go
reader := bytes.NewReader(data)
doc, err := godocx.OpenReader(reader, int64(len(data)))
```

The reader must stay valid for the lifetime of the `Document`, because image
bytes are read on demand.

Load an embedded image lazily:

```go
image, err := doc.Image("rId5")
if err != nil {
	return err
}
fmt.Println(image.ContentType, image.Filename, len(image.Bytes))
```

## Text Output

`Document.ToText()` emits a plain-text linearization:

- Paragraphs are separated by newlines.
- Lists render with indentation and prefixes, for example `- `, `1. `, `a. `.
- Tables render as tab-separated rows.
- Vertically merged table continuations render the inherited origin text.
- Horizontally merged cells render their text once and leave covered columns empty.
- Images render as `[image: <filename>]`.
- Footnote and endnote references render as `[footnote: <id>]` and `[endnote: <id>]`.

## Markdown and Chunks

`Document.ToMarkdown()` emits Markdown for headers, body, footers, footnotes,
and endnotes. Links, basic run formatting, images, tables, and note definitions
are represented in Markdown-friendly form for ingestion pipelines.

`Document.Chunks()` returns ordered RAG units with stable source/path metadata:
`Source`, `SourceID`, `Path`, `Markdown`, `List`, `Table`, `Image`, and `Notes`.
The earlier `Kind`, `Level`, `Text`, `TableID`, and `ImageID` fields remain
available for callers using the preliminary API.

## Testing Strategy

Tests synthesize minimal DOCX archives in memory with hand-written XML. This
keeps the unit tests focused on OOXML edge cases without depending on large
binary fixtures.

The real-world service contract fixture is under `testdata/`; see
`testdata/README.md` for its coverage and the generated fixture list.

## Verification

Use the project commands before submitting changes:

```bash
make tidy
make test
make lint
make build
go vet ./...
go test ./... -race
```

`make tidy` must be used instead of running `go mod tidy` directly in normal
workflow.

## Benchmarks

`benchmark_test.go` measures parsing (`Open`, `OpenReader`), rendering
(`ToText`, `ToMarkdown`, `Chunks`), and lazy image loading on the `testdata`
fixtures, plus scaling on synthetic documents with 100, 1,000, and 10,000
blocks.

```bash
make bench
make bench BENCH=Chunks BENCH_COUNT=10
```

Use `BENCH_COUNT=10` with `benchstat` to compare results before and after a
change.
