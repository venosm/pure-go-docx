// Package godocx reads Microsoft Word DOCX files without external dependencies.
//
// The package extracts document text, tables, images, and structural metadata
// from OOXML packages. It does not render documents or support legacy binary
// .doc files.
package godocx

import (
	"fmt"
	"io"
	"os"
)

// Open opens a .docx file from disk.
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

	return OpenReader(file, stat.Size())
}

// OpenReader opens a .docx from any io.ReaderAt with known size.
func OpenReader(r io.ReaderAt, size int64) (*Document, error) {
	return openReader(r, size)
}
