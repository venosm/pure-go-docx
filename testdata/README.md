# DOCX Fixture Plan

The unit tests synthesize focused DOCX archives in memory. Before merging with
real procurement samples, place representative fixtures here:

- `simple.docx`: ordinary paragraphs, headings, tabs, and line breaks.
- `tables.docx`: flat tables with multiple rows and columns.
- `nested-tables.docx`: recursive tables inside cells.
- `lists-nested.docx`: numbered and bulleted lists with nested levels.
- `images.docx`: embedded PNG/JPEG/GIF/SVG images with alt text.
- `czech-diacritics.docx`: Czech UTF-8 text with preserved run whitespace.
- `headers-footers.docx`: headers, footers, and relationships.
- `sdt-content-controls.docx`: structured document tags with body content.
- `malformed-truncated.docx`: damaged or truncated archive/XML for robustness tests.
