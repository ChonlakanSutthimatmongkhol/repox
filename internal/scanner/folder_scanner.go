package scanner

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
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

// DetectFeatureStructure inspects subdirectories recursively (up to depth 3) of featureRoot
// and returns "clean_architecture", "grouped", or "flat".
func DetectFeatureStructure(featureRoot string) (string, error) {
	cleanPatternHits := 0
	groupedPatternHits := 0

	err := filepath.WalkDir(featureRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || !d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(featureRoot, path)
		if err != nil || rel == "." {
			return nil
		}

		depth := strings.Count(rel, string(filepath.Separator))
		if depth > 2 {
			return filepath.SkipDir
		}

		if excludedDirs[d.Name()] {
			return filepath.SkipDir
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		hasPresentation := false
		hasDomain := false
		hasData := false
		hasBloc := false
		hasScreen := false
		hasRepository := false

		for _, sub := range entries {
			if !sub.IsDir() {
				continue
			}
			switch sub.Name() {
			case "presentation":
				hasPresentation = true
			case "domain":
				hasDomain = true
			case "data":
				hasData = true
			case "bloc":
				hasBloc = true
			case "screen":
				hasScreen = true
			case "repository":
				hasRepository = true
			}
		}

		if hasPresentation && (hasDomain || hasData) {
			cleanPatternHits++
		}
		if hasBloc || hasScreen || hasRepository {
			groupedPatternHits++
		}

		return nil
	})
	if err != nil {
		return "flat", nil
	}

	if cleanPatternHits > 0 {
		return "clean_architecture", nil
	}
	if groupedPatternHits > 0 {
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

// AnalyzeAllFeatures scans all feature directories and returns their info and detected patterns.
func AnalyzeAllFeatures(featureRoot string) ([]models.FeatureInfo, error) {
	var features []models.FeatureInfo

	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		featurePath := filepath.Join(featureRoot, entry.Name())
		structure := detectSingleFeatureStructure(featurePath)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		features = append(features, models.FeatureInfo{
			Name:         entry.Name(),
			Structure:    structure,
			LastModified: info.ModTime().Format(time.RFC3339),
			Path:         featurePath,
		})
	}

	return features, nil
}

// detectSingleFeatureStructure checks a single feature directory and returns its structure type.
func detectSingleFeatureStructure(featurePath string) string {
	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return "flat"
	}

	hasPresentation := false
	hasDomain := false
	hasData := false
	hasBloc := false
	hasScreen := false
	hasRepository := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		switch entry.Name() {
		case "presentation":
			hasPresentation = true
		case "domain":
			hasDomain = true
		case "data":
			hasData = true
		case "bloc":
			hasBloc = true
		case "screen":
			hasScreen = true
		case "repository":
			hasRepository = true
		}
	}

	if hasPresentation && (hasDomain || hasData) {
		return "clean_architecture"
	}
	if hasBloc || hasScreen || hasRepository {
		return "grouped"
	}
	return "flat"
}
