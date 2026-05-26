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
- Streaming XML parsing with `encoding/xml.Decoder.Token`.
- Paragraphs, runs, tabs, line breaks, headings, and basic run formatting flags.
- Numbered and bulleted list resolution with nesting and ordinals.
- Flat and nested tables.
- Table `gridSpan` expansion into dense grids.
- Table `vMerge` continuation inheritance so each row is self-contained.
- Embedded image references and lazy image byte loading.
- Plain-text output through `Document.ToText()`.
- Synthetic DOCX unit tests built in memory.

Not implemented yet:

- Headers, footers, footnotes, and endnotes.
- Full Markdown output for RAG ingestion.
- Production-shaped chunk metadata beyond the current preliminary API.
- `mc:AlternateContent` choice/fallback handling.
- Field code display-value state handling.
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

Open from any random-access reader:

```go
reader := bytes.NewReader(data)
doc, err := godocx.OpenReader(reader, int64(len(data)))
```

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

## Testing Strategy

Tests synthesize minimal DOCX archives in memory with hand-written XML. This
keeps the unit tests focused on OOXML edge cases without depending on large
binary fixtures.

Real-world fixtures can be added under `testdata/`; see
`testdata/README.md` for the planned coverage list.

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
