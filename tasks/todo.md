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

## Kroky (Thin Slices)

### Slice 1: Skeleton and Test Helper
- [x] Implementace
- [x] Test
- [x] Verifikace: `make test`

### Slice 2: OPC Loading
- [x] Implementace
- [x] Test
- [x] Verifikace: `make test`

### Slice 3: Body Parser
- [x] Implementace
- [x] Test
- [x] Verifikace: `make test`

### Slice 4: Public API and Rendering
- [x] Implementace
- [x] Test
- [x] Verifikace: `make test`

### Milestone 2 Task 1: Numbering Resolver
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test ./internal/numbering`

### Milestone 2 Task 2: Numbering Wire-Up
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test -run 'TestParagraph_' .`

### Milestone 2 Task 3: Table Merge Resolution
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test -run 'TestTable_' .`

### Milestone 2 Task 4: ToText Rendering
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test -run 'TestToText_|TestFormatOrdinal' .`

### Milestone 3 Task 1: Parser Support
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test ./...`

### Milestone 3 Task 2: Related DOCX Parts
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test ./...`

### Milestone 3 Task 3: Markdown and Chunks
- [x] Implementace
- [x] Test
- [x] Verifikace: `go test ./...`

## Checkpoints
- [2026-05-24] Rozhodnuti: projekt je samostatny Go modul v adresari `pure-go-docx`.
- [2026-05-25] Rozhodnuti: module path je `github.com/venosm/pure-go-docx` a Go directive je `go 1.26`.
- [2026-05-24] Zjisteni: Go `encoding/xml` stream parser handles DOCX namespace URIs correctly when switching by `Name.Local` after root validation.
- [2026-05-25] Rozhodnuti: Milestone 2 zacina izolovane balickem `internal/numbering`; parser tela dokumentu zatim zustava bez zmen.
- [2026-05-25] Zjisteni: `numFmt="none"` musi vynutit prazdny `LevelText`, aby pozdejsi renderer nevytvoril prefix.
- [2026-05-25] Rozhodnuti: numbering part se nacita pres relationship target hlavniho dokumentu, ne pres pevnou cestu `word/numbering.xml`.
- [2026-05-25] Rozhodnuti: horizontalni merge se v dense gridu reprezentuje origin cell + prazdne pokryte sloupce, aby se text neopakoval.
- [2026-05-25] Zjisteni: vMerge continuation sdili `Blocks` s originem, aby kazdy radek tabulky byl samostatne smysluplny pro RAG.
- [2026-05-26] Zjisteni: parser zatim akceptuje jen `w:document/w:body`; milestone 3 vyzaduje znovupouzitelne parsovani rootu `w:hdr`, `w:ftr`, `w:footnotes` a `w:endnotes`.
- [2026-05-26] Rozhodnuti: `ToText` zustane kompatibilni jako linearizace hlavniho tela, zatimco `ToMarkdown` a `Chunks` zahrnou souvisejici casti pro RAG.
- [2026-05-26] Rozhodnuti: obrazky v hlavnim tele zustavaji dostupne pres puvodni `rId`, obrazky v souvisejicich castech pouzivaji part-aware ID ve tvaru `word/header1.xml#rIdImage`.
- [2026-05-26] Zjisteni: `mc:AlternateContent` muze obalit bloky, run obsah i kresby; parser vybira prvni `Choice` a pri jeho absenci `Fallback`.

## Vysledky

### Zmenene soubory
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

### Verifikace
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
