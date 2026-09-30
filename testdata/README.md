# DOCX Fixtures

Two kinds of fixtures live here:

- Generated English fixtures, produced by `testdata/gen` and covering one
  parser feature area each.
- Real-world Czech procurement samples, used by `TestRealDocxFixtures`.

The unit tests in `internal/` still synthesize focused DOCX archives in memory.
The fixtures below cover the packaging layer that in-memory tests skip: content
types, relationships, related parts, and media.

## Generated English Fixtures

Regenerate every file from the repository root:

```bash
go run ./testdata/gen
```

The archives are byte-stable, so regenerating without source changes leaves the
working tree clean. List the fixtures without writing files with
`go run ./testdata/gen -list`.

| Fixture | Coverage |
|---|---|
| `simple.docx` | Paragraphs, `Heading1`–`Heading3`, tabs, line breaks, bold/italic/underline runs, external and anchor hyperlinks. |
| `tables.docx` | Two flat tables with a bold header row and multiple data rows. |
| `tables-merged.docx` | `gridSpan` expansion and `vMerge` continuation inheritance. |
| `nested-tables.docx` | A table nested inside a cell of another table. |
| `lists-nested.docx` | `numbering.xml` with decimal/lowerLetter/lowerRoman levels, a bulleted list, and a `startOverride` that restarts numbering at five. |
| `images.docx` | PNG, JPEG, GIF, and SVG images with English alt text, including one image repeated inside a table cell. |
| `unicode-whitespace.docx` | English text with diacritics, currency, CJK and Arabic characters, and `xml:space="preserve"` run whitespace. |
| `headers-footers.docx` | Default and first-page headers, one footer, and an image relationship owned by the header part. |
| `footnotes-endnotes.docx` | Footnote and endnote definitions with in-body references, including a reference inside a table cell. Separator notes are present and must be ignored. |
| `sdt-content-controls.docx` | Structured document tags wrapping paragraphs, headings, a whole table, and a table cell. |
| `fields-alternate-content.docx` | Complex fields, nested fields, `w:fldSimple`, `HYPERLINK` field links, and `mc:AlternateContent` choice/fallback at block and run level. |
| `service-agreement-en.docx` | Full English service agreement combining every feature above: 40 body blocks, 2 headers, 1 footer, 3 footnotes, numbering, merged tables, and images in both the body and a header. |
| `malformed-truncated.docx` | A valid package cut short so the zip central directory is missing. `Open` must return an error. |

### Image identity

`images.docx` and `service-agreement-en.docx` both reference `rId1` from a
header part and another relationship from the body, so part-scoped image IDs
(`word/header2.xml#rId1`) are exercised rather than only bare relationship IDs.

## Real-World Czech Fixtures

- `Priloha c. 1 - Formular nabidky - PROHLASENI DODAVATELE - vzor.docx`
- `Priloha c. 2 – Smlouva – vzor.docx`

These are Word-authored procurement documents. They cover markup that
hand-written fixtures do not reproduce faithfully, such as Word's own style and
numbering definitions.
