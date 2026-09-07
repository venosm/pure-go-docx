package godocx

import "github.com/venosm/pure-go-docx/internal/body"

// Block is one ordered top-level or nested document block.
type Block = body.Block

// Paragraph is a Word paragraph with style metadata and ordered runs.
type Paragraph = body.Paragraph

// ListRef describes a numbered or bulleted list item.
type ListRef = body.ListRef

// Run is a paragraph run containing text, formatting, links, or an image.
type Run = body.Run

// NoteRef is a footnote or endnote reference found in a run.
type NoteRef = body.NoteRef

const (
	// NoteKindFootnote identifies a footnote reference or chunk source.
	NoteKindFootnote = body.NoteKindFootnote
	// NoteKindEndnote identifies an endnote reference or chunk source.
	NoteKindEndnote = body.NoteKindEndnote
)

// Table is a rectangular table grid.
type Table = body.Table

// Cell is one table cell. Nested paragraphs and tables are stored in Blocks.
type Cell = body.Cell

// MergeKind identifies vertical table merge behavior.
type MergeKind = body.MergeKind

const (
	// MergeNone means the table cell is not vertically merged.
	MergeNone = body.MergeNone
	// MergeRestart means the table cell starts a vertical merge.
	MergeRestart = body.MergeRestart
	// MergeContinue means the table cell continues a vertical merge.
	MergeContinue = body.MergeContinue
)

// Image is an embedded document image loaded from the DOCX package.
type Image = body.Image

// ImageRef is a lazy reference to an embedded image found in a run.
type ImageRef = body.ImageRef

// Chunk is an ordered RAG ingestion unit derived from document blocks.
type Chunk = body.Chunk

// ChunkList describes list metadata for a paragraph chunk.
type ChunkList = body.ChunkList

// ChunkTable describes table position metadata for a chunk.
type ChunkTable = body.ChunkTable

// ChunkImage describes image metadata for a chunk.
type ChunkImage = body.ChunkImage
