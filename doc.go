// Package godocx reads Microsoft Word DOCX files without external dependencies.
//
// The package extracts document text, tables, images, and structural metadata
// from OOXML packages. It does not render documents or support legacy binary
// .doc files.
package godocx

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// maxArchiveBytes omezuje buffer archivu, který Open drží v paměti.
//
// Hodnota odpovídá limitu nekomprimovaného obsahu v internal/opc. Archiv
// s nekomprimovanými položkami může být o zip hlavičky větší než jeho obsah,
// takže Open může odmítnout soubor těsně pod limitem, který by OpenReader
// přijal. U dokumentů blízko 500 MiB je to přijatelná odchylka.
const maxArchiveBytes = 500 << 20

// Open opens a .docx file from disk.
//
// The archive is buffered in memory, so the returned Document stays usable
// after Open returns. This keeps lazy image loading through Document.Image and
// Document.AllImages working: only the requested media part is decompressed,
// on demand.
//
// Callers that must avoid the buffer, for example by memory-mapping the file
// or by keeping the file handle open themselves, should use OpenReader with
// their own io.ReaderAt and keep it valid for the lifetime of the Document.
func Open(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening DOCX file %q: %w", path, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat DOCX file %q: %w", path, err)
	}
	if stat.Size() > maxArchiveBytes {
		return nil, fmt.Errorf("DOCX file %q exceeds limit: %d > %d", path, stat.Size(), maxArchiveBytes)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading DOCX file %q: %w", path, err)
	}

	return OpenReader(bytes.NewReader(data), int64(len(data)))
}

// OpenReader opens a .docx from any io.ReaderAt with known size.
//
// The reader must stay valid for the lifetime of the returned Document,
// because image bytes are read lazily.
func OpenReader(r io.ReaderAt, size int64) (*Document, error) {
	return openReader(r, size)
}
