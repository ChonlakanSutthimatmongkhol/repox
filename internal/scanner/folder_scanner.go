package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

var flutterFeatureRootCandidates = []string{
	"lib/features",
	"lib/feature",
	"lib/modules",
	"lib/pages",
	"lib/screens",
}

var flutterTestRootCandidates = []string{
	"test/features",
	"test/feature",
	"test/unit",
	"test",
}

// DetectFeatureRoot returns the existing feature root with the most feature folders.
// Returns an empty string (no error) if none is found.
func DetectFeatureRoot(rootDir, projectType string) (string, error) {
	var candidates []string
	switch projectType {
	case "flutter", "dart":
		candidates = flutterFeatureRootCandidates
	case "go":
		candidates = []string{"internal", "pkg", "cmd"}
	default:
		return "", nil
	}

	best := ""
	bestCount := -1
	for _, c := range candidates {
		full := filepath.Join(rootDir, c)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			count := countFeatureFolders(full)
			if best == "" || count > bestCount {
				best = c
				bestCount = count
			}
		}
	}
	return best, nil
}

// DetectFeatureStructure inspects the first-level subdirectories of featureRoot
// and returns "clean_architecture", "grouped", or "flat".
func DetectFeatureStructure(featureRoot string) (string, error) {
	analysis, err := AnalyzeFeatureRoot(filepath.Dir(featureRoot), filepath.Base(featureRoot))
	if err == nil && analysis.RecommendedPattern != "" {
		return analysis.RecommendedPattern, nil
	}
	return detectFeatureRootPatternFallback(featureRoot), nil
}

// DetectTestRoot returns the first existing test root directory for the project type.
func DetectTestRoot(rootDir, projectType string) (string, error) {
	var candidates []string
	switch projectType {
	case "flutter", "dart":
		candidates = flutterTestRootCandidates
	default:
		// Go convention: tests live alongside source
		return "", nil
	}

	for _, c := range candidates {
		full := filepath.Join(rootDir, c)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "test", nil
}

func countFeatureFolders(featureRoot string) int {
	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && !excludedDirs[entry.Name()] {
			count++
		}
	}
	return count
}

func detectFeatureRootPatternFallback(featureRoot string) string {
	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return "flat"
	}

	counts := map[string]int{}
	for _, entry := range entries {
		if !entry.IsDir() || excludedDirs[entry.Name()] || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		counts[DetectFeaturePattern(filepath.Join(featureRoot, entry.Name()))]++
	}
	if pattern := recommendedPattern(counts); pattern != "" {
		return pattern
	}
	return DetectFeaturePattern(featureRoot)
}
