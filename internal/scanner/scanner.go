package scanner

import (
	"fmt"
	"os"
	"path/filepath"
)

// scanDirectory scans a given directory (and its subdirectories) for files
// and returns a list of file paths found within the directory and subdirectories.
func ScanDirectory(rootDir string) ([]string, error) {
	var files []string

	// Walk through the directory and its subdirectories
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only add regular files (ignore directories)
		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %v", err)
	}

	return files, nil
}
