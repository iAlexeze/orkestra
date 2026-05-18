//go:build !runtime

package cli

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// writeOutput writes data to either a file or a directory.
// If path is empty → prints to stdout.
// If path is a directory → writes <path>/<filename>.
// If path is a file → writes directly to that file.
func writeOutput(path, filename string, data []byte) error {
	// No output flag → print to stdout
	if path == "" {
		fmt.Println(string(data))
		return nil
	}

	// Check if path exists
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// Path is a directory → write <dir>/<filename>
		full := filepath.Join(path, filename)
		log.Printf("%s generated successfully\n", filename)
		return os.WriteFile(full, data, 0644)
	}

	// Otherwise treat as file path
	log.Printf("%s generated successfully\n", filename)
	return os.WriteFile(path, data, 0644)
}
