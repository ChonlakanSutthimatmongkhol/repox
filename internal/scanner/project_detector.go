package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// DetectProjectType returns "flutter", "dart", "go", "node", or "unknown"
// based on marker files found in rootDir.
func DetectProjectType(rootDir string) (string, error) {
	pubspec := filepath.Join(rootDir, "pubspec.yaml")
	if _, err := os.Stat(pubspec); err == nil {
		data, err := os.ReadFile(pubspec)
		if err != nil {
			return "dart", nil
		}
		if strings.Contains(string(data), "flutter:") {
			return "flutter", nil
		}
		return "dart", nil
	}

	if _, err := os.Stat(filepath.Join(rootDir, "go.mod")); err == nil {
		return "go", nil
	}

	if _, err := os.Stat(filepath.Join(rootDir, "package.json")); err == nil {
		return "node", nil
	}

	return "unknown", nil
}
