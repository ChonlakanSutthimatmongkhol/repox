package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var (
	dartClassRe  = regexp.MustCompile(`(?m)\bclass\s+([A-Za-z_]\w*)(?:\s*<[^>{}]*>)?(?:\s+extends\s+([A-Za-z_]\w*(?:<[^>{}]*>)?))?(?:\s+with\s+([^{]+?))?(?:\s+implements\s+([^{]+?))?\s*\{`)
	dartMethodRe = regexp.MustCompile(`^(?:Future(?:<[^>]+>)?|Stream(?:<[^>]+>)?|void|Widget|bool|String|int|double|[A-Za-z_]\w*(?:<[^;{=]*>)?)\s+([A-Za-z_]\w*)\s*\(`)
	goFuncRe     = regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(`)
	goTypeRe     = regexp.MustCompile(`(?m)^type\s+([A-Za-z_]\w*)\s+(?:struct|interface)\b`)
	depTypeRe    = regexp.MustCompile(`\b([A-Z][A-Za-z0-9_]*(?:UseCase|Repository|Repo|Service|Analytics|Bloc|Cubit|Client|DataSource))\b`)
	getItTypeRe  = regexp.MustCompile(`(?:getIt|locator|serviceLocator|sl)\s*<\s*([A-Z][A-Za-z0-9_]*)\s*>`)
)

func collectFeatureAnatomy(rootDir string, files map[string]string) map[string]models.FileAnatomy {
	anatomy := map[string]models.FileAnatomy{}
	for role, relPath := range files {
		fullPath := filepath.Join(rootDir, relPath)
		item := analyzeFileAnatomy(role, filepath.ToSlash(relPath), fullPath)
		if hasAnatomy(item) {
			anatomy[role] = item
		}
	}
	return anatomy
}

func analyzeFileAnatomy(role, relPath, fullPath string) models.FileAnatomy {
	item := models.FileAnatomy{
		Role: role,
		Path: relPath,
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return item
	}
	content := string(data)

	switch filepath.Ext(fullPath) {
	case ".dart":
		item.Imports, _ = parseDartImports(fullPath)
		item.ClassNames, item.BaseClasses, item.Mixins = parseDartClasses(content)
		item.Methods = parseDartMethods(content)
		item.ConstructorDeps = parseDependencyTypes(content)
	case ".go":
		item.Imports, _ = parseGoImports(fullPath)
		item.ClassNames = parseGoTypes(content)
		item.Methods = parseGoFuncs(content)
		item.ConstructorDeps = parseDependencyTypes(content)
	}

	item.Capabilities = detectCapabilities(relPath, content)
	item.HasFirebaseTracking = contains(item.Capabilities, "firebase_tracking")
	item.ConstructorDeps = removeKnownTypes(item.ConstructorDeps, item.ClassNames, item.BaseClasses, item.Mixins)
	return dedupFileAnatomy(item)
}

func hasAnatomy(item models.FileAnatomy) bool {
	return len(item.ClassNames) > 0 ||
		len(item.BaseClasses) > 0 ||
		len(item.Mixins) > 0 ||
		len(item.Methods) > 0 ||
		len(item.ConstructorDeps) > 0 ||
		len(item.Imports) > 0 ||
		len(item.Capabilities) > 0
}

func parseDartClasses(content string) ([]string, []string, []string) {
	var classes, bases, mixins []string
	for _, match := range dartClassRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 && match[1] != "" {
			classes = append(classes, match[1])
		}
		if len(match) > 2 && match[2] != "" {
			bases = append(bases, cleanTypeName(match[2]))
		}
		if len(match) > 3 && match[3] != "" {
			for _, mixin := range splitTypeList(match[3]) {
				mixins = append(mixins, cleanTypeName(mixin))
			}
		}
	}
	return dedup(classes), dedup(bases), dedup(mixins)
}

func parseDartMethods(content string) []string {
	var methods []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "@") || strings.HasPrefix(line, "//") {
			continue
		}
		match := dartMethodRe.FindStringSubmatch(line)
		if len(match) < 2 || match[1] == "" || startsUpper(match[1]) {
			continue
		}
		methods = append(methods, match[1])
	}
	return dedup(methods)
}

func parseGoTypes(content string) []string {
	var types []string
	for _, match := range goTypeRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			types = append(types, match[1])
		}
	}
	return dedup(types)
}

func parseGoFuncs(content string) []string {
	var funcs []string
	for _, match := range goFuncRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 || match[1] == "" {
			continue
		}
		funcs = append(funcs, match[1])
	}
	return dedup(funcs)
}

func parseDependencyTypes(content string) []string {
	var deps []string
	for _, match := range depTypeRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			deps = append(deps, cleanTypeName(match[1]))
		}
	}
	for _, match := range getItTypeRe.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			deps = append(deps, cleanTypeName(match[1]))
		}
	}
	return dedup(deps)
}

func detectCapabilities(path, content string) []string {
	lowerPath := strings.ToLower(path)
	lowerContent := strings.ToLower(content)
	var capabilities []string
	if strings.Contains(lowerPath, "firebase") ||
		strings.Contains(lowerContent, "firebaseanalytics") ||
		strings.Contains(lowerContent, "logevent") ||
		strings.Contains(lowerContent, "trackscreen") ||
		strings.Contains(lowerContent, "trackscreenview") {
		capabilities = append(capabilities, "firebase_tracking")
	}
	if strings.Contains(lowerPath, "analytics") ||
		strings.Contains(lowerContent, "analytics") ||
		strings.Contains(lowerContent, "track") {
		capabilities = append(capabilities, "analytics")
	}
	if strings.Contains(lowerContent, "basebloc") || strings.Contains(lowerContent, "base_bloc") {
		capabilities = append(capabilities, "base_bloc")
	}
	if strings.Contains(lowerContent, "baseroute") || strings.Contains(lowerContent, "route_model") {
		capabilities = append(capabilities, "route_model")
	}
	return dedup(capabilities)
}

func buildRoleAnatomy(features []models.FeatureAnalysis) map[string]models.RoleAnatomy {
	type roleVotes struct {
		total            int
		baseClasses      map[string]int
		mixins           map[string]int
		methods          map[string]int
		constructorDeps  map[string]int
		imports          map[string]int
		capabilities     map[string]int
		firebaseTracking int
	}

	votes := map[string]*roleVotes{}
	for _, feature := range features {
		for role, item := range feature.Anatomy {
			rv := votes[role]
			if rv == nil {
				rv = &roleVotes{
					baseClasses:     map[string]int{},
					mixins:          map[string]int{},
					methods:         map[string]int{},
					constructorDeps: map[string]int{},
					imports:         map[string]int{},
					capabilities:    map[string]int{},
				}
				votes[role] = rv
			}
			rv.total++
			addVotes(rv.baseClasses, item.BaseClasses)
			addVotes(rv.mixins, item.Mixins)
			addVotes(rv.methods, item.Methods)
			addVotes(rv.constructorDeps, item.ConstructorDeps)
			addVotes(rv.imports, item.Imports)
			addVotes(rv.capabilities, item.Capabilities)
			if item.HasFirebaseTracking {
				rv.firebaseTracking++
			}
		}
	}

	result := map[string]models.RoleAnatomy{}
	for role, rv := range votes {
		result[role] = models.RoleAnatomy{
			FeatureCount:        rv.total,
			BaseClasses:         anatomyVotes(rv.baseClasses, rv.total, 20, 10),
			Mixins:              anatomyVotes(rv.mixins, rv.total, 20, 10),
			Methods:             anatomyVotes(rv.methods, rv.total, 25, 15),
			ConstructorDeps:     anatomyVotes(rv.constructorDeps, rv.total, 20, 15),
			Imports:             anatomyVotes(rv.imports, rv.total, 30, 15),
			Capabilities:        anatomyVotes(rv.capabilities, rv.total, 20, 10),
			HasFirebaseTracking: anatomyVote("firebase_tracking", rv.firebaseTracking, rv.total),
		}
	}
	return result
}

func addVotes(freq map[string]int, items []string) {
	for _, item := range dedup(items) {
		if item != "" {
			freq[item]++
		}
	}
}

func anatomyVotes(freq map[string]int, total, minPercentage, limit int) []models.AnatomyVote {
	votes := make([]models.AnatomyVote, 0, len(freq))
	for name, count := range freq {
		vote := anatomyVote(name, count, total)
		if vote.Percentage >= float64(minPercentage) {
			votes = append(votes, vote)
		}
	}
	sort.Slice(votes, func(i, j int) bool {
		if votes[i].Count == votes[j].Count {
			return votes[i].Name < votes[j].Name
		}
		return votes[i].Count > votes[j].Count
	})
	if limit > 0 && len(votes) > limit {
		return votes[:limit]
	}
	return votes
}

func anatomyVote(name string, count, total int) models.AnatomyVote {
	vote := models.AnatomyVote{Name: name, Count: count}
	if total > 0 {
		vote.Percentage = float64(count) * 100 / float64(total)
	}
	return vote
}

func splitTypeList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '&'
	})
}

func cleanTypeName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.Index(value, "<"); idx >= 0 {
		value = value[:idx]
	}
	value = strings.Trim(value, " \t\r\n,{}()?")
	return value
}

func dedupFileAnatomy(item models.FileAnatomy) models.FileAnatomy {
	item.ClassNames = dedup(item.ClassNames)
	item.BaseClasses = dedup(item.BaseClasses)
	item.Mixins = dedup(item.Mixins)
	item.Methods = dedup(item.Methods)
	item.ConstructorDeps = dedup(item.ConstructorDeps)
	item.Imports = dedup(item.Imports)
	item.Capabilities = dedup(item.Capabilities)
	return item
}

func removeKnownTypes(items []string, knownGroups ...[]string) []string {
	known := map[string]bool{}
	for _, group := range knownGroups {
		for _, item := range group {
			known[item] = true
		}
	}
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if !known[item] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func dedup(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

func startsUpper(value string) bool {
	if value == "" {
		return false
	}
	r := rune(value[0])
	return r >= 'A' && r <= 'Z'
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
