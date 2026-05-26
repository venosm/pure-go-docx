package body

// Block is one ordered top-level or nested document block.
type Block interface {
	blockKind() string
}

// Paragraph is a Word paragraph with style metadata and ordered runs.
type Paragraph struct {
	StyleID    string
	HeadingLvl int
	List       *ListRef
	Runs       []Run
}

func (*Paragraph) blockKind() string { return "paragraph" }

// ListRef describes a numbered or bulleted list item.
type ListRef struct {
	NumID     int
	Level     int
	Format    string
	LevelText string
	Ordinal   int
}

// Run is a paragraph run containing text, formatting, links, or an image.
type Run struct {
	Text                    string
	Bold, Italic, Underline bool
	Tab                     bool
	Break                   bool
	Image                   *ImageRef
	Link                    string
}

// Table is a rectangular table grid.
type Table struct {
	Grid [][]Cell
}

func (*Table) blockKind() string { return "table" }

// Cell is one table cell. Nested paragraphs and tables are stored in Blocks.
type Cell struct {
	Blocks []Block
	HSpan  int
	VMerge MergeKind
}

// MergeKind identifies vertical table merge behavior.
type MergeKind int

const (
	// MergeNone means the table cell is not vertically merged.
	MergeNone MergeKind = iota
	// MergeRestart means the table cell starts a vertical merge.
	MergeRestart
	// MergeContinue means the table cell continues a vertical merge.
	MergeContinue
)

// Image is an embedded document image loaded from the DOCX package.
type Image struct {
	RelID       string
	ContentType string
	Filename    string
	Bytes       []byte
	AltText     string
}

// ImageRef is a lazy reference to an embedded image found in a run.
type ImageRef struct {
	RelID               string
	Filename            string
	AltText             string
	WidthEMU, HeightEMU int64
}

// Chunk is an ordered RAG ingestion unit derived from document blocks.
type Chunk struct {
	Kind    string
	Level   int
	Text    string
	TableID int
	ImageID string
}
