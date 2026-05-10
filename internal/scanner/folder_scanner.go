package scanner

import (
	"os"
	"path/filepath"
	"strings"

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

// DetectFeatureRoot returns the existing feature root with the strongest feature signal.
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
	bestScore := -1
	for _, c := range candidates {
		full := filepath.Join(rootDir, c)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			count, score := scoreFeatureRoot(full, c, projectType)
			if best == "" || score > bestScore || (score == bestScore && count > bestCount) {
				best = c
				bestCount = count
				bestScore = score
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
	count, _ := scoreFeatureRoot(featureRoot, "", "")
	return count
}

func scoreFeatureRoot(featureRoot, candidate, projectType string) (int, int) {
	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return 0, 0
	}
	count := 0
	score := 0
	for _, entry := range entries {
		if !entry.IsDir() || excludedDirs[entry.Name()] {
			continue
		}
		// Only count directories that look like features (have role files or
		// internal sub-packages like handler/, service/, repository/).
		// Using empty naming → bootstrap pattern detection.
		featurePath := filepath.Join(featureRoot, entry.Name())
		if isFeatureUnit(featurePath, models.NamingConvention{}) {
			count++
			score += 10
			if projectType == "go" {
				score += scoreGoFeatureUnit(featurePath)
			}
		}
	}
	if projectType == "go" {
		score += scoreGoFeatureRootCandidate(candidate)
	}
	return count, score
}

func scoreGoFeatureRootCandidate(candidate string) int {
	switch candidate {
	case "internal":
		return 5
	case "cmd":
		return -20
	default:
		return 0
	}
}

func scoreGoFeatureUnit(featurePath string) int {
	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return 0
	}
	roleDirs := 0
	for _, entry := range entries {
		if entry.IsDir() && isGoFeatureRoleDir(entry.Name()) {
			roleDirs++
		}
	}
	switch {
	case roleDirs >= 3:
		return 80 + roleDirs*10
	case roleDirs == 2:
		return 50
	case roleDirs == 1:
		return 8
	default:
		return 0
	}
}

func isGoFeatureRoleDir(name string) bool {
	switch name {
	case "handler", "handlers", "controller", "controllers",
		"service", "services", "repository", "repositories",
		"usecase", "usecases", "dto", "entity", "entities",
		"errors", "domain", "data", "delivery":
		return true
	default:
		return false
	}
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
