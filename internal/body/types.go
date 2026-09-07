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
	Note                    *NoteRef
	Link                    string
}

// NoteRef is a footnote or endnote reference found in a run.
type NoteRef struct {
	Kind string
	ID   string
}

const (
	// NoteKindFootnote identifies a footnote reference or chunk source.
	NoteKindFootnote = "footnote"
	// NoteKindEndnote identifies an endnote reference or chunk source.
	NoteKindEndnote = "endnote"
)

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
	ID          string
	PartName    string
	RelID       string
	ContentType string
	Filename    string
	Bytes       []byte
	AltText     string
}

// ImageRef is a lazy reference to an embedded image found in a run.
type ImageRef struct {
	ID                  string
	PartName            string
	RelID               string
	Filename            string
	AltText             string
	WidthEMU, HeightEMU int64
}

// Chunk is an ordered RAG ingestion unit derived from document blocks.
type Chunk struct {
	ID       string
	Index    int
	Kind     string
	Source   string
	SourceID string
	Path     string
	Level    int
	Text     string
	Markdown string
	StyleID  string
	List     *ChunkList
	Table    *ChunkTable
	Image    *ChunkImage
	Notes    []NoteRef

	// TableID and ImageID are retained for callers using the preliminary API.
	TableID int
	ImageID string
}

// ChunkList describes list metadata for a paragraph chunk.
type ChunkList struct {
	NumID     int
	Level     int
	Format    string
	LevelText string
	Ordinal   int
	Marker    string
}

// ChunkTable describes table position metadata for a chunk.
type ChunkTable struct {
	ID      int
	Row     int
	Column  int
	Columns int
}

// ChunkImage describes image metadata for a chunk.
type ChunkImage struct {
	ID        string
	PartName  string
	RelID     string
	Filename  string
	AltText   string
	WidthEMU  int64
	HeightEMU int64
}
