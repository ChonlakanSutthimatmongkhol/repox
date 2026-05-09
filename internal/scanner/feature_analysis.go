package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var knownFeaturePatterns = []string{"flat", "grouped", "clean_architecture"}

// AnalyzeFeatureRoot enumerates every feature folder under featureRoot and
// analyzes the structure pattern used by each feature.
func AnalyzeFeatureRoot(rootDir, featureRoot string) (models.FeaturesAnalysis, error) {
	analysis := models.FeaturesAnalysis{
		Features:            []models.FeatureAnalysis{},
		PatternDistribution: emptyPatternDistribution(),
	}
	if featureRoot == "" {
		return analysis, nil
	}

	featureRootPath := filepath.Join(rootDir, featureRoot)
	entries, err := os.ReadDir(featureRootPath)
	if err != nil {
		return analysis, nil
	}

	var latestFeature models.FeatureAnalysis
	var latestTime time.Time
	counts := map[string]int{}

	for _, entry := range entries {
		if !entry.IsDir() || excludedDirs[entry.Name()] || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		featurePath := filepath.Join(featureRootPath, entry.Name())
		structure := DetectFeaturePattern(featurePath)
		lastModified := latestModified(featurePath)
		relPath := filepath.ToSlash(filepath.Join(featureRoot, entry.Name()))

		feature := models.FeatureAnalysis{
			Name:         entry.Name(),
			Path:         relPath,
			Structure:    structure,
			LastModified: lastModified.UTC().Format(time.RFC3339),
		}
		analysis.Features = append(analysis.Features, feature)
		counts[structure]++

		if lastModified.After(latestTime) {
			latestTime = lastModified
			latestFeature = feature
		}
	}

	sort.Slice(analysis.Features, func(i, j int) bool {
		return analysis.Features[i].Path < analysis.Features[j].Path
	})

	analysis.PatternDistribution = buildPatternDistribution(counts, len(analysis.Features))
	analysis.RecommendedPattern = recommendedPattern(counts)
	analysis.LatestPattern = latestFeature.Structure
	return analysis, nil
}

// DetectFeaturePattern checks one feature directory recursively and classifies
// it as flat, grouped, or clean_architecture.
func DetectFeaturePattern(featurePath string) string {
	dirs := map[string]bool{}
	_ = filepath.WalkDir(featurePath, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != featurePath && (excludedDirs[name] || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		if path != featurePath {
			dirs[name] = true
		}
		return nil
	})

	if dirs["presentation"] && (dirs["domain"] || dirs["data"]) {
		return "clean_architecture"
	}
	if dirs["bloc"] || dirs["screen"] || dirs["screens"] || dirs["repository"] || dirs["repositories"] {
		return "grouped"
	}
	return "flat"
}

func emptyPatternDistribution() map[string]models.PatternDistribution {
	distribution := make(map[string]models.PatternDistribution, len(knownFeaturePatterns))
	for _, pattern := range knownFeaturePatterns {
		distribution[pattern] = models.PatternDistribution{}
	}
	return distribution
}

func buildPatternDistribution(counts map[string]int, total int) map[string]models.PatternDistribution {
	distribution := emptyPatternDistribution()
	if total == 0 {
		return distribution
	}
	for _, pattern := range knownFeaturePatterns {
		count := counts[pattern]
		distribution[pattern] = models.PatternDistribution{
			Count:      count,
			Percentage: float64(count) * 100 / float64(total),
		}
	}
	return distribution
}

func recommendedPattern(counts map[string]int) string {
	bestPattern := ""
	bestCount := 0
	for _, pattern := range knownFeaturePatterns {
		if counts[pattern] > bestCount {
			bestPattern = pattern
			bestCount = counts[pattern]
		}
	}
	return bestPattern
}

func latestModified(root string) time.Time {
	var latest time.Time
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != root && (excludedDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	if latest.IsZero() {
		if info, err := os.Stat(root); err == nil {
			return info.ModTime()
		}
		return time.Time{}
	}
	return latest
}
