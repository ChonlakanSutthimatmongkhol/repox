// Package learner compares generated scaffolds with developer edits and extracts lessons.
package learner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// DiffResult holds the comparison between generated and current file.
type DiffResult struct {
	Path          string
	GeneratedCode string
	CurrentCode   string
	Diff          string
	Changed       bool
}

// ReadDiffs compares the snapshot of a generation with the current files on disk.
func ReadDiffs(generation models.Generation, baseDir string) ([]DiffResult, error) {
	if generation.SnapshotDir == "" {
		return nil, fmt.Errorf("learner: generation %q has no snapshot (was it generated with --ai?)", generation.ID)
	}

	var results []DiffResult
	for _, relPath := range generation.Files {
		snapPath := filepath.Join(generation.SnapshotDir, relPath)
		currentPath := filepath.Join(baseDir, relPath)

		generatedCode, err := os.ReadFile(snapPath)
		if err != nil {
			continue // snapshot missing — skip
		}
		currentCode, err := os.ReadFile(currentPath)
		if err != nil {
			continue // file was deleted — skip
		}

		gen := string(generatedCode)
		cur := string(currentCode)
		changed := gen != cur

		diff := ""
		if changed {
			diff = unifiedDiff(relPath, gen, cur)
		}

		results = append(results, DiffResult{
			Path:          relPath,
			GeneratedCode: gen,
			CurrentCode:   cur,
			Diff:          diff,
			Changed:       changed,
		})
	}
	return results, nil
}

// unifiedDiff produces a simple line-by-line unified diff.
func unifiedDiff(path, original, modified string) string {
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)

	// Simple diff: show removed then added lines (not a true LCS diff, but sufficient for AI consumption)
	maxLen := len(origLines)
	if len(modLines) > maxLen {
		maxLen = len(modLines)
	}

	for i := 0; i < maxLen; i++ {
		origLine := ""
		modLine := ""
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(modLines) {
			modLine = modLines[i]
		}
		if origLine != modLine {
			if origLine != "" {
				fmt.Fprintf(&b, "-%s\n", origLine)
			}
			if modLine != "" {
				fmt.Fprintf(&b, "+%s\n", modLine)
			}
		} else {
			fmt.Fprintf(&b, " %s\n", origLine)
		}
	}
	return b.String()
}
