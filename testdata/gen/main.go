// Command gen writes the English-language DOCX fixtures used by the
// repository tests.
//
// Run it from the repository root:
//
//	go run ./testdata/gen
//
// The generated archives are byte-stable, so re-running the command without
// source changes leaves the working tree clean.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "testdata", "directory that receives the generated fixtures")
	list := flag.Bool("list", false, "print the fixture list without writing files")
	flag.Parse()

	if err := run(*outDir, *list); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}

func run(outDir string, list bool) error {
	all := fixtures()
	if list {
		for _, f := range all {
			fmt.Printf("%-32s %s\n", f.name, f.description)
		}
		return nil
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory %q: %w", outDir, err)
	}

	for _, f := range all {
		data, err := f.build()
		if err != nil {
			return fmt.Errorf("building %s: %w", f.name, err)
		}
		path := filepath.Join(outDir, f.name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", path, len(data))
	}
	return nil
}
