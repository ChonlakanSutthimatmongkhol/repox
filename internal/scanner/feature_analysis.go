package scanner

import (
	"fmt"
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
func AnalyzeFeatureRoot(rootDir, featureRoot string, naming ...models.NamingConvention) (models.FeaturesAnalysis, error) {
	n := models.NamingConvention{}
	if len(naming) > 0 {
		n = naming[0]
	}
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

	counts := map[string]int{}
	if len(entries) == 0 {
		return analysis, nil
	}

	features, latestFeature := discoverFeatureAnalyses(rootDir, featureRootPath, featureRoot, n)
	for _, feature := range features {
		analysis.Features = append(analysis.Features, feature)
		counts[feature.Structure]++
	}

	sort.Slice(analysis.Features, func(i, j int) bool {
		return analysis.Features[i].Path < analysis.Features[j].Path
	})

	analysis.PatternDistribution = buildPatternDistribution(counts, len(analysis.Features))
	analysis.RecommendedPattern = recommendedPattern(counts)
	analysis.LatestPattern = latestFeature.Structure
	analysis.RoleAnatomy = buildRoleAnatomy(analysis.Features)
	return analysis, nil
}

// InferPatternMappings learns role routes from scanned feature files.
func InferPatternMappings(features []models.FeatureAnalysis, fallback models.PatternMappings) models.PatternMappings {
	mappings := clonePatternMappings(fallback)
	routeCounts := map[string]map[string]map[string]int{}

	for _, feature := range features {
		pattern := feature.Structure
		if pattern == "" || len(feature.FileRoutes) == 0 {
			continue
		}
		if routeCounts[pattern] == nil {
			routeCounts[pattern] = map[string]map[string]int{}
		}
		for role, route := range feature.FileRoutes {
			if role == "" {
				continue
			}
			if routeCounts[pattern][role] == nil {
				routeCounts[pattern][role] = map[string]int{}
			}
			routeCounts[pattern][role][route]++
		}
	}

	for pattern, byRole := range routeCounts {
		mapping := mappings[pattern]
		if mapping.FileRoutes == nil {
			mapping.FileRoutes = map[string]string{}
		}
		for role, counts := range byRole {
			mapping.FileRoutes[role] = mostCommonRoute(counts)
		}
		mappings[pattern] = mapping
	}
	return mappings
}

func clonePatternMappings(in models.PatternMappings) models.PatternMappings {
	out := models.PatternMappings{}
	for pattern, mapping := range in {
		copied := models.PatternMapping{FileRoutes: map[string]string{}}
		for role, route := range mapping.FileRoutes {
			copied.FileRoutes[role] = route
		}
		out[pattern] = copied
	}
	return out
}

func mostCommonRoute(counts map[string]int) string {
	bestRoute := ""
	bestCount := -1
	for route, count := range counts {
		if count > bestCount || (count == bestCount && route < bestRoute) {
			bestRoute = route
			bestCount = count
		}
	}
	return bestRoute
}

// cleanArchLayers are the universal DDD/Clean-Architecture layer names.
// They are architectural conventions shared across all projects, not team-specific.
var cleanArchLayers = map[string]bool{
	"presentation": true, "domain": true, "data": true,
	"infrastructure": true, "application": true,
}

// bootstrapInternalDirNames is the fallback used before naming conventions are scanned.
var bootstrapInternalDirNames = []string{
	"bloc", "cubit", "screen", "screens", "page", "pages",
	"repository", "repositories", "usecase", "usecases",
	"model", "models", "widget", "widgets", "delivery",
	"handler", "handlers", "controller", "controllers",
	"service", "services", "request", "requests",
	"response", "responses", "enum", "enums", "firebase", "analytics",
}

// internalDirSet builds the set of directories that are role containers or
// architecture layers rather than nested feature roots. Bootstrap role folders
// keep generic model/request/response layouts discoverable; scanned suffix roles
// add project-specific role folders on top.
func internalDirSet(naming models.NamingConvention) map[string]bool {
	set := map[string]bool{}
	for layer := range cleanArchLayers {
		set[layer] = true
	}
	for _, name := range bootstrapInternalDirNames {
		set[name] = true
	}
	if len(naming.SuffixRoles) > 0 {
		for _, role := range naming.SuffixRoles {
			set[role] = true
			set[role+"s"] = true // common plural form
		}
		return set
	}
	return set
}

// DetectFeaturePattern checks one feature directory and classifies it as
// flat, grouped, or clean_architecture.
// Pass naming so that role-dir detection is driven by scanned conventions
// rather than hardcoded strings.
func DetectFeaturePattern(featurePath string, naming ...models.NamingConvention) string {
	n := models.NamingConvention{}
	if len(naming) > 0 {
		n = naming[0]
	}

	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return "flat"
	}

	dirs := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if excludedDirs[name] || strings.HasPrefix(name, ".") {
			continue
		}
		dirs[name] = true
	}

	if dirs["presentation"] && (dirs["domain"] || dirs["data"]) {
		return "clean_architecture"
	}
	groupedSet := internalDirSet(n)
	for dir := range dirs {
		if groupedSet[dir] {
			return "grouped"
		}
	}
	return "flat"
}

func discoverFeatureAnalyses(rootDir, featureRootPath, featureRoot string, naming models.NamingConvention) ([]models.FeatureAnalysis, models.FeatureAnalysis) {
	var features []models.FeatureAnalysis
	var latestFeature models.FeatureAnalysis
	var latestTime time.Time

	_ = filepath.WalkDir(featureRootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != featureRootPath && (excludedDirs[name] || strings.HasPrefix(name, ".")) {
			return filepath.SkipDir
		}
		if path == featureRootPath {
			return nil
		}
		if isFeatureInternalDir(name, naming) {
			return filepath.SkipDir
		}
		if !isFeatureUnit(path, naming) {
			return nil
		}

		relUnderRoot, err := filepath.Rel(featureRootPath, path)
		if err != nil {
			return nil
		}
		relUnderRoot = filepath.ToSlash(relUnderRoot)
		relPath := filepath.ToSlash(filepath.Join(featureRoot, relUnderRoot))
		parent := filepath.ToSlash(filepath.Dir(relUnderRoot))
		if parent == "." {
			parent = ""
		}
		files, routes := collectFeatureFiles(rootDir, path, naming)
		if len(files) == 0 {
			return nil
		}
		anatomy := collectFeatureAnatomy(rootDir, files)
		lastModified := latestModified(path)
		feature := models.FeatureAnalysis{
			Name:         filepath.Base(path),
			Path:         relPath,
			Parent:       parent,
			Depth:        featureDepth(relUnderRoot),
			Structure:    DetectFeaturePattern(path, naming),
			LastModified: lastModified.UTC().Format(time.RFC3339),
			FileCount:    len(files),
			Files:        files,
			FileRoutes:   routes,
			Anatomy:      anatomy,
		}
		features = append(features, feature)

		if lastModified.After(latestTime) {
			latestTime = lastModified
			latestFeature = feature
		}
		return nil
	})

	return features, latestFeature
}

func isFeatureUnit(path string, naming models.NamingConvention) bool {
	if isFeatureInternalDir(filepath.Base(path), naming) {
		return false
	}
	if DetectFeaturePattern(path, naming) != "flat" {
		return true
	}
	if hasFeatureRoleFilesUnderInternalDirs(path, naming) {
		return true
	}
	return hasImmediateFeatureRoleFile(path, naming)
}

func hasImmediateFeatureRoleFile(path string, naming models.NamingConvention) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if roleForFeatureFile(entry.Name(), naming) != "" {
			return true
		}
	}
	return false
}

func hasFeatureRoleFilesUnderInternalDirs(path string, naming models.NamingConvention) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() || !isFeatureInternalDir(entry.Name(), naming) {
			continue
		}
		found := false
		_ = filepath.WalkDir(filepath.Join(path, entry.Name()), func(childPath string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if childPath != filepath.Join(path, entry.Name()) && (excludedDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if roleForFeatureFile(d.Name(), naming) != "" {
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		if found {
			return true
		}
	}
	return false
}

type featureFileCandidate struct {
	rel   string
	role  string
	route string
	depth int
}

func collectFeatureFiles(rootDir, featureDir string, naming models.NamingConvention) (map[string]string, map[string]string) {
	var candidates []featureFileCandidate

	_ = filepath.WalkDir(featureDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != featureDir && (excludedDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
				return filepath.SkipDir
			}
			if path != featureDir && !isFeatureInternalDir(d.Name(), naming) && isFeatureUnit(path, naming) {
				return filepath.SkipDir
			}
			return nil
		}
		if isGeneratedSourceFile(path) {
			return nil
		}
		role := roleForFeaturePath(featureDir, path, naming)
		if role == "" {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}
		route, err := filepath.Rel(featureDir, filepath.Dir(path))
		if err != nil || route == "." {
			route = ""
		}
		depth := strings.Count(filepath.ToSlash(rel), "/")
		candidates = append(candidates, featureFileCandidate{
			rel:   filepath.ToSlash(rel),
			role:  role,
			route: filepath.ToSlash(route),
			depth: depth,
		})
		return nil
	})

	// Sort shallowest files first so primary roles (screen, bloc, etc.) go to
	// files closest to the feature root, not nested subdirectory variants.
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].depth != candidates[j].depth {
			return candidates[i].depth < candidates[j].depth
		}
		return candidates[i].rel < candidates[j].rel
	})

	files := map[string]string{}
	routes := map[string]string{}
	for _, c := range candidates {
		role := uniqueFeatureRole(c.role, c.route, files)
		files[role] = c.rel
		routes[role] = c.route
	}
	return files, routes
}

func isGeneratedSourceFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".g.dart") ||
		strings.HasSuffix(base, ".freezed.dart") ||
		strings.HasSuffix(base, ".gen.dart") ||
		strings.HasSuffix(base, ".generated.dart")
}

func roleForFeaturePath(featureDir, path string, naming models.NamingConvention) string {
	if role := roleForFeatureFile(filepath.Base(path), naming); role != "" {
		return role
	}
	ext := filepath.Ext(path)
	if ext != ".dart" && ext != ".go" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), ext)
	featureName := filepath.Base(featureDir)
	if strings.HasPrefix(base, featureName+"_") {
		base = strings.TrimPrefix(base, featureName+"_")
	}
	role := strings.Trim(base, "_")
	if role == "" || role == featureName {
		return ""
	}
	return role
}

func uniqueFeatureRole(role, route string, files map[string]string) string {
	if _, exists := files[role]; !exists {
		return role
	}
	prefix := strings.NewReplacer("/", "_", "\\", "_").Replace(route)
	prefix = strings.Trim(prefix, "_")
	if prefix == "" {
		prefix = role
	}
	candidate := prefix + "_" + role
	for i := 2; ; i++ {
		if _, exists := files[candidate]; !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", prefix+"_"+role, i)
	}
}

// roleForFeatureFile maps a filename to a repox role.
// Uses naming.SuffixRoles (built by the scanner from detected suffixes) first.
// Falls back to bootstrap patterns for generic roles that are not suffix-voted.
func roleForFeatureFile(filename string, naming models.NamingConvention) string {
	ext := filepath.Ext(filename)
	if ext != ".dart" && ext != ".go" {
		return ""
	}
	base := strings.ToLower(strings.TrimSuffix(filename, ext))

	if len(naming.SuffixRoles) > 0 {
		// Lookup from scan-built map — longest suffix wins to handle "repositoryimpl" before "repository".
		bestRole, bestLen := "", 0
		for suffix, role := range naming.SuffixRoles {
			if len(suffix) > bestLen && strings.HasSuffix(base, "_"+suffix) {
				bestRole, bestLen = role, len(suffix)
			}
		}
		if bestRole != "" {
			return bestRole
		}
	}

	// Bootstrap fallback — same candidates the naming scanner votes on, plus
	// generic model/request/response roles needed to discover scanned metadata.
	switch {
	case strings.HasSuffix(base, "_repository_impl"):
		return "repository_impl"
	case strings.HasSuffix(base, "_screen"), strings.HasSuffix(base, "_page"), strings.HasSuffix(base, "_view"):
		return "screen"
	case strings.HasSuffix(base, "_bloc"), strings.HasSuffix(base, "_cubit"):
		return "bloc"
	case strings.HasSuffix(base, "_event"):
		return "event"
	case strings.HasSuffix(base, "_state"):
		return "state"
	case strings.HasSuffix(base, "_repository"), strings.HasSuffix(base, "_repo"):
		return "repository"
	case strings.HasSuffix(base, "_usecase"), strings.HasSuffix(base, "_use_case"):
		return "usecase"
	case strings.HasSuffix(base, "_request"), strings.HasSuffix(base, "_request_model"):
		return "request"
	case strings.HasSuffix(base, "_response"), strings.HasSuffix(base, "_response_model"):
		return "response"
	case strings.HasSuffix(base, "_handler"), strings.HasSuffix(base, "_controller"):
		return "handler"
	case strings.HasSuffix(base, "_service"):
		return "service"
	default:
		return ""
	}
}

func featureDepth(relUnderRoot string) int {
	if relUnderRoot == "" || relUnderRoot == "." {
		return 0
	}
	return len(strings.Split(filepath.ToSlash(relUnderRoot), "/"))
}

func isFeatureInternalDir(name string, naming ...models.NamingConvention) bool {
	n := models.NamingConvention{}
	if len(naming) > 0 {
		n = naming[0]
	}
	return internalDirSet(n)[name]
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
