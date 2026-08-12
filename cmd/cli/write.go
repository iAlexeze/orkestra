//go:build !runtime && !gateway

package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// writeOutput writes data to either a file, a directory, or stdout.
// If path is "-"  → prints to stdout.
// If path is ""   → writes <filename> in the current directory.
// If path is a directory → writes <path>/<filename>.
// If path is a file → writes directly to that file.
func writeOutput(path, filename string, data []byte) error {
	if path == "-" {
		fmt.Println(string(data))
		return nil
	}

	var dest string
	if path == "" {
		dest = filename
	} else {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			dest = filepath.Join(path, filename)
		} else {
			dest = path
		}
	}

	log.Printf("%s generated successfully\n", filepath.Base(dest))
	return os.WriteFile(dest, data, 0644)
}
