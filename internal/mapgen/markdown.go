// Package mapgen builds deterministic markdown maps from scanned conventions.
package mapgen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// GeneratedMap is a map file path and content pair.
type GeneratedMap struct {
	Path    string
	Content string
}

// GenerateProjectMarkdown returns a human-readable project map.
func GenerateProjectMarkdown(conv models.Convention) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Repox Project Map")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Project type: %s\n", valueOrUnknown(conv.ProjectType))
	fmt.Fprintf(&b, "- Feature root: %s\n", valueOrUnknown(conv.FeatureRoot))
	fmt.Fprintf(&b, "- Test root: %s\n", valueOrUnknown(conv.TestRoot))
	fmt.Fprintf(&b, "- Recommended pattern: %s\n", recommendedPattern(conv))
	fmt.Fprintf(&b, "- Features indexed: %d\n", len(conv.FeaturesAnalysis.Features))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Features")
	features := sortedFeatures(conv.FeaturesAnalysis.Features)
	if len(features) == 0 {
		fmt.Fprintln(&b, "- No features indexed yet.")
		return strings.TrimSpace(b.String()) + "\n"
	}
	for _, feature := range features {
		roles := sortedMapKeys(feature.Files)
		if len(roles) == 0 {
			fmt.Fprintf(&b, "- %s\n", feature.Path)
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", feature.Path, strings.Join(roles, ", "))
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// GenerateConventionsMarkdown returns a convention-focused map.
func GenerateConventionsMarkdown(conv models.Convention) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Repox Convention Map")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Project type: %s\n", valueOrUnknown(conv.ProjectType))
	fmt.Fprintf(&b, "- State management: %s\n", valueOrUnknown(conv.StateManagement))
	fmt.Fprintf(&b, "- Routing: %s", valueOrUnknown(conv.Routing.Type))
	if conv.Routing.RouteFile != "" {
		fmt.Fprintf(&b, " (%s)", conv.Routing.RouteFile)
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Recommended pattern: %s\n", recommendedPattern(conv))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Roles")
	roles := sortedMapKeys(conv.Roles)
	if len(roles) == 0 {
		roles = sortedRoleAnatomyKeys(conv.FeaturesAnalysis.RoleAnatomy)
	}
	if len(roles) == 0 {
		fmt.Fprintln(&b, "- No roles detected yet.")
	} else {
		for _, role := range roles {
			fmt.Fprintf(&b, "- %s\n", role)
		}
	}
	if len(conv.CommonImports) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Common Imports")
		for _, item := range firstN(conv.CommonImports, 10) {
			fmt.Fprintf(&b, "- `%s`\n", item)
		}
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// GenerateProjectMarkmap returns markmap-compatible markdown.
func GenerateProjectMarkmap(conv models.Convention) string {
	var b strings.Builder
	fmt.Fprintln(&b, "# Repox Map")
	fmt.Fprintln(&b, "## Project")
	fmt.Fprintf(&b, "- Type: %s\n", valueOrUnknown(conv.ProjectType))
	fmt.Fprintf(&b, "- Feature root: %s\n", valueOrUnknown(conv.FeatureRoot))
	fmt.Fprintf(&b, "- Pattern: %s\n", recommendedPattern(conv))
	fmt.Fprintln(&b, "## Features")
	for _, feature := range sortedFeatures(conv.FeaturesAnalysis.Features) {
		fmt.Fprintf(&b, "- %s\n", feature.Path)
		for _, role := range sortedMapKeys(feature.Files) {
			fmt.Fprintf(&b, "  - %s\n", role)
		}
	}
	return strings.TrimSpace(b.String()) + "\n"
}

// GenerateFeatureMarkdown returns a focused map for one feature.
func GenerateFeatureMarkdown(conv models.Convention, query string) (string, string, bool) {
	feature, ok := FindFeature(conv, query)
	if !ok {
		return "", "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Feature Map: %s\n\n", feature.Path)
	fmt.Fprintf(&b, "- Name: %s\n", feature.Name)
	fmt.Fprintf(&b, "- Structure: %s\n", valueOrUnknown(feature.Structure))
	fmt.Fprintf(&b, "- Files: %d\n", len(feature.Files))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Files")
	for _, role := range sortedMapKeys(feature.Files) {
		fmt.Fprintf(&b, "- %s: `%s`\n", role, feature.Files[role])
	}
	return featureSafeName(conv, feature), strings.TrimSpace(b.String()) + "\n", true
}

// GenerateFeatureMarkmap returns markmap-compatible markdown for one feature.
func GenerateFeatureMarkmap(conv models.Convention, query string) (string, string, bool) {
	feature, ok := FindFeature(conv, query)
	if !ok {
		return "", "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n", feature.Path)
	fmt.Fprintln(&b, "## Files")
	for _, role := range sortedMapKeys(feature.Files) {
		fmt.Fprintf(&b, "- %s\n", role)
		fmt.Fprintf(&b, "  - %s\n", feature.Files[role])
	}
	return featureSafeName(conv, feature), strings.TrimSpace(b.String()) + "\n", true
}

// FindFeature locates a feature by absolute-ish path, feature-root relative path, or name.
func FindFeature(conv models.Convention, query string) (models.FeatureAnalysis, bool) {
	query = filepath.ToSlash(strings.Trim(query, "/ "))
	base := filepath.Base(query)
	root := filepath.ToSlash(strings.Trim(conv.FeatureRoot, "/"))
	for _, feature := range conv.FeaturesAnalysis.Features {
		path := filepath.ToSlash(strings.Trim(feature.Path, "/"))
		rel := path
		if root != "" && strings.HasPrefix(path, root+"/") {
			rel = strings.TrimPrefix(path, root+"/")
		}
		if path == query || rel == query || feature.Name == query || feature.Name == base {
			return feature, true
		}
	}
	return models.FeatureAnalysis{}, false
}

func sortedFeatures(features []models.FeatureAnalysis) []models.FeatureAnalysis {
	cp := append([]models.FeatureAnalysis(nil), features...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Path < cp[j].Path })
	return cp
}

func sortedMapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedRoleAnatomyKeys(m map[string]models.RoleAnatomy) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstN(items []string, n int) []string {
	if len(items) < n {
		n = len(items)
	}
	return items[:n]
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func recommendedPattern(conv models.Convention) string {
	if conv.FeaturesAnalysis.RecommendedPattern != "" {
		return conv.FeaturesAnalysis.RecommendedPattern
	}
	return valueOrUnknown(conv.FeatureStructure)
}

func featureSafeName(conv models.Convention, feature models.FeatureAnalysis) string {
	name := filepath.ToSlash(strings.Trim(feature.Path, "/"))
	parent := strings.Trim(filepath.ToSlash(feature.Parent), "/")
	if parent != "" && strings.HasPrefix(name, parent+"/") {
		name = strings.TrimPrefix(name, parent+"/")
	}
	root := strings.Trim(filepath.ToSlash(conv.FeatureRoot), "/")
	if root != "" && strings.HasPrefix(name, root+"/") {
		name = strings.TrimPrefix(name, root+"/")
	}
	name = strings.ReplaceAll(name, "/", "_")
	if name == "" || name == "." {
		name = feature.Name
	}
	return name
}
