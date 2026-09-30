# Task: pure-go-docx DOCX reader

## Status
- [x] Milestone 1 Completed
- [x] Milestone 2 Task 1 Completed
- [x] Milestone 2 Task 2 Completed
- [x] Milestone 2 Completed
- [x] Milestone 3 Completed

## Acceptance Criteria
- [x] Milestone 1 parses synthetic DOCX paragraphs, flat tables, and embedded images
- [x] Public API compiles with documented exported identifiers
- [x] Standard-library-only dependency policy is preserved
- [x] Verification passes: `make tidy`, `make test`, `make lint`, `make build`, `go vet ./...`
- [x] Milestone 2 Task 1 parses numbering.xml and resolves list ordinals
- [x] Milestone 2 Task 2 populates `Paragraph.List` from `w:numPr`
- [x] Milestone 2 Task 3 resolves table gridSpan and vMerge into dense grids
- [x] Milestone 2 Task 4 renders lists and merged tables through `ToText`
- [x] Milestone 3 parses headers, footers, footnotes, and endnotes
- [x] Milestone 3 renders full Markdown for body and related DOCX parts
- [x] Milestone 3 returns production-shaped chunk metadata
- [x] Milestone 3 handles `mc:AlternateContent` choice/fallback wrappers
- [x] Milestone 3 renders field display values without field instructions
- [x] Verification passes: `make tidy`, `make test`, `make lint`, `make build`, `go vet ./...`

## Steps (Thin Slices)

### Slice 1: Skeleton and Test Helper
- [x] Implementation
- [x] Test
- [x] Verification: `make test`

### Slice 2: OPC Loading
- [x] Implementation
- [x] Test
- [x] Verification: `make test`

### Slice 3: Body Parser
- [x] Implementation
- [x] Test
- [x] Verification: `make test`

### Slice 4: Public API and Rendering
- [x] Implementation
- [x] Test
- [x] Verification: `make test`

### Milestone 2 Task 1: Numbering Resolver
- [x] Implementation
- [x] Test
- [x] Verification: `go test ./internal/numbering`

### Milestone 2 Task 2: Numbering Wire-Up
- [x] Implementation
- [x] Test
- [x] Verification: `go test -run 'TestParagraph_' .`

### Milestone 2 Task 3: Table Merge Resolution
- [x] Implementation
- [x] Test
- [x] Verification: `go test -run 'TestTable_' .`

### Milestone 2 Task 4: ToText Rendering
- [x] Implementation
- [x] Test
- [x] Verification: `go test -run 'TestToText_|TestFormatOrdinal' .`

### Milestone 3 Task 1: Parser Support
- [x] Implementation
- [x] Test
- [x] Verification: `go test ./...`

### Milestone 3 Task 2: Related DOCX Parts
- [x] Implementation
- [x] Test
- [x] Verification: `go test ./...`

### Milestone 3 Task 3: Markdown and Chunks
- [x] Implementation
- [x] Test
- [x] Verification: `go test ./...`

## Checkpoints
- [2026-05-24] Decision: the project is a standalone Go module in the `pure-go-docx` directory.
- [2026-05-25] Decision: the module path is `github.com/venosm/pure-go-docx` and the Go directive is `go 1.26`.
- [2026-05-24] Finding: Go's `encoding/xml` stream parser handles DOCX namespace URIs correctly when switching by `Name.Local` after root validation.
- [2026-05-25] Decision: Milestone 2 starts with the isolated `internal/numbering` package; the document body parser remains unchanged for now.
- [2026-05-25] Finding: `numFmt="none"` must force an empty `LevelText` so the renderer does not add a prefix.
- [2026-05-25] Decision: load the numbering part through the main document's relationship target, instead of a fixed `word/numbering.xml` path.
- [2026-05-25] Decision: represent a horizontal merge in the dense grid as the origin cell and empty covered columns so text is not repeated.
- [2026-05-25] Finding: a vMerge continuation shares `Blocks` with the origin so each table row is independently useful for RAG.
- [2026-05-26] Finding: the parser currently accepts only `w:document/w:body`; Milestone 3 requires reusable parsing of `w:hdr`, `w:ftr`, `w:footnotes`, and `w:endnotes` roots.
- [2026-05-26] Decision: keep `ToText` compatible as a linear rendering of the main body, while `ToMarkdown` and `Chunks` include related parts for RAG.
- [2026-05-26] Decision: images in the main body remain accessible by their original `rId`; images in related parts use part-aware IDs such as `word/header1.xml#rIdImage`.
- [2026-05-26] Finding: `mc:AlternateContent` can wrap blocks, run content, and drawings; the parser chooses the first `Choice` or `Fallback` if no choice exists.

## Results

### Changed Files
- `doc.go`, `document.go`, `types.go`, `render.go` - public API, lazy image loading, text/markdown/chunk rendering.
- `internal/opc/` - OPC zip abstraction, content types, relationships, size limits.
- `internal/body/` - streaming body parser for paragraphs, runs, tables, SDT content, and drawing image references.
- `internal/testutil/` - synthetic DOCX builder for focused unit tests.
- `internal/numbering/` - Milestone 2 Task 1 resolver for numbering.xml definitions and counters.
- `paragraph_test.go` - Milestone 2 Task 2 numbering wire-up tests.
- `table_test.go` - Milestone 2 Task 3 dense table and merge tests.
- `render_test.go` - Milestone 2 Task 4 list/table text rendering tests.
- `testdata/README.md` - fixture plan for real-world integration documents.
- `document.go`, `types.go`, `render.go` - Milestone 3 related parts, note/endnote maps, part-aware image API, Markdown, chunk metadata.
- `internal/body/` - Milestone 3 reusable root parsers, note references, alternate content, field display-value state, drawing choice handling.
- `internal/opc/rels.go` - Milestone 3 relationship type constants for headers, footers, footnotes, and endnotes.
- `document_test.go`, `paragraph_test.go`, `render_test.go`, `internal/testutil/docx.go` - Milestone 3 synthetic DOCX regressions and helpers.
- `README.md` - Milestone 3 implemented status and API documentation.

### Verification
- `make tidy` - PASS
- `make test` - PASS
- `make lint` - PASS
- `make build` - PASS
- `go vet ./...` - PASS
- `go test ./internal/numbering` - PASS
- `go test ./... -race` - PASS
- `go test -run 'TestParagraph_' .` - PASS
- `go test -run 'TestTable_' .` - PASS
- `go test -run 'TestToText_|TestFormatOrdinal' .` - PASS
- `go test -bench=. ./...` - PASS
- `make tidy` - PASS (Milestone 3)
- `make test` - PASS (Milestone 3)
- `make lint` - PASS (Milestone 3)
- `make build` - PASS (Milestone 3)
- `go vet ./...` - PASS (Milestone 3)
- `go test ./... -race` - PASS (Milestone 3)

### Verification Story
Milestone 2 was verified with synthetic DOCX archives covering numbering
relationships, list ordinal state, gridSpan expansion, vMerge inheritance,
nested tables, malformed orphan continuations, list prefixes, ordinal formats,
and merged table text output.

Milestone 3 was verified with synthetic DOCX archives covering related header,
footer, footnote, and endnote parts; note references; related-part images;
block/run/drawing alternate content; complex and simple field display values;
Markdown output; and chunk source/path/list/table/image/note metadata.
