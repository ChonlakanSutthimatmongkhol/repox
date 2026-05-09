package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteResult records the outcome of writing a single generated file.
type WriteResult struct {
	Path    string
	Written bool
	Skipped bool
	Reason  string
}

// WriteFiles writes generated files to baseDir, skipping existing files unless force is true.
func WriteFiles(files []GeneratedFile, baseDir string, force bool) ([]WriteResult, error) {
	results := make([]WriteResult, 0, len(files))

	for _, f := range files {
		dest := filepath.Join(baseDir, f.Path)

		if !force {
			if _, err := os.Stat(dest); err == nil {
				results = append(results, WriteResult{
					Path:    dest,
					Skipped: true,
					Reason:  fmt.Sprintf("File exists: %s. Use --force to overwrite", dest),
				})
				continue
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return results, fmt.Errorf("file_writer: mkdir %s: %w", filepath.Dir(dest), err)
		}

		if err := os.WriteFile(dest, []byte(f.Content), 0o644); err != nil {
			return results, fmt.Errorf("file_writer: write %s: %w", dest, err)
		}

		results = append(results, WriteResult{Path: dest, Written: true})
	}
	return results, nil
}
