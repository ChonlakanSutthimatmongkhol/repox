package scanner

import (
	"os"
	"path/filepath"
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

// DetectFeatureRoot returns the first existing feature root directory for the project type.
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

	for _, c := range candidates {
		full := filepath.Join(rootDir, c)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", nil
}

// DetectFeatureStructure inspects the first-level subdirectories of featureRoot
// and returns "clean_architecture", "grouped", or "flat".
func DetectFeatureStructure(featureRoot string) (string, error) {
	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return "flat", nil
	}

	cleanArch := map[string]bool{"presentation": false, "domain": false, "data": false}
	grouped := map[string]bool{"bloc": false, "screen": false, "repository": false}

	for _, entry := range entries {
		if !entry.IsDir() || excludedDirs[entry.Name()] {
			continue
		}
		featureDir := filepath.Join(featureRoot, entry.Name())
		subs, err := os.ReadDir(featureDir)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			if !sub.IsDir() {
				continue
			}
			name := sub.Name()
			if _, ok := cleanArch[name]; ok {
				cleanArch[name] = true
			}
			if _, ok := grouped[name]; ok {
				grouped[name] = true
			}
		}
	}

	cleanCount := 0
	for _, v := range cleanArch {
		if v {
			cleanCount++
		}
	}
	if cleanCount >= 2 {
		return "clean_architecture", nil
	}
	if grouped["bloc"] || grouped["screen"] || grouped["repository"] {
		return "grouped", nil
	}
	return "flat", nil
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
