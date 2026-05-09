package retriever

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var indexExcludedDirs = map[string]bool{
	".repox":       true,
	"build":        true,
	".dart_tool":   true,
	"node_modules": true,
	".git":         true,
}

// FeatureIndexer implements Retriever using the local filesystem.
type FeatureIndexer struct{}

func (f *FeatureIndexer) Index(rootDir string, conv *models.Convention) ([]models.Example, error) {
	return IndexFeatures(rootDir, conv)
}

func (f *FeatureIndexer) FindSimilar(target string, examples []models.Example, topN int) []models.Example {
	return FindSimilar(target, examples, topN)
}

// IndexFeatures walks the first-level subdirectories of featureRoot and builds
// an Example for each feature found.
func IndexFeatures(rootDir string, conv *models.Convention) ([]models.Example, error) {
	featureRoot := filepath.Join(rootDir, conv.FeatureRoot)
	entries, err := os.ReadDir(featureRoot)
	if err != nil {
		return nil, nil // feature root doesn't exist yet — not an error
	}

	var examples []models.Example
	for _, entry := range entries {
		if !entry.IsDir() || indexExcludedDirs[entry.Name()] {
			continue
		}
		featureDir := filepath.Join(featureRoot, entry.Name())
		ex := buildExample(rootDir, featureDir, entry.Name(), conv)
		examples = append(examples, ex)
	}

	sort.Slice(examples, func(i, j int) bool {
		return examples[i].Name < examples[j].Name
	})
	return examples, nil
}

func buildExample(rootDir, featureDir, name string, conv *models.Convention) models.Example {
	if conv.ProjectType == "go" {
		return buildGoExample(rootDir, featureDir, name, conv)
	}
	return buildFlutterExample(rootDir, featureDir, name, conv)
}

// ── Flutter ───────────────────────────────────────────────────────────────────

func buildFlutterExample(rootDir, featureDir, name string, conv *models.Convention) models.Example {
	relPath, _ := filepath.Rel(rootDir, featureDir)

	files := map[string]string{}
	var allImports []string

	_ = filepath.Walk(featureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".dart") {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)
		base := filepath.Base(path)

		switch {
		case hasSuffix(base, conv.Naming.ScreenSuffix):
			files["screen"] = rel
		case hasSuffix(base, conv.Naming.BlocSuffix):
			files["bloc"] = rel
		case hasSuffix(base, conv.Naming.EventSuffix):
			files["event"] = rel
		case hasSuffix(base, conv.Naming.StateSuffix):
			files["state"] = rel
		case hasSuffix(base, conv.Naming.RepositorySuffix):
			if _, exists := files["repository"]; !exists {
				files["repository"] = rel
			} else {
				files["repository_impl"] = rel
			}
		case hasSuffix(base, conv.Naming.UsecaseSuffix):
			files["usecase"] = rel
		}

		allImports = append(allImports, parseDartImports(path)...)
		return nil
	})

	uniqueImports := dedup(allImports)
	structure := detectFlutterStructure(featureDir)
	patterns := deriveFlutterPatterns(uniqueImports)

	meta := models.FeatureMetadata{
		HasBloc:       files["bloc"] != "",
		HasScreen:     files["screen"] != "",
		HasRepository: files["repository"] != "",
		HasUseCase:    files["usecase"] != "",
		HasTest:       hasFlutterTestFiles(rootDir, name),
		Imports:       uniqueImports,
		Structure:     structure,
	}

	return models.Example{
		Name:     name,
		Path:     relPath,
		Files:    files,
		Patterns: patterns,
		Metadata: meta,
	}
}

// hasSuffix reports whether the dart filename (without extension) ends with suffix (PascalCase converted to snake).
func hasSuffix(filename, suffix string) bool {
	if suffix == "" {
		return false
	}
	name := strings.TrimSuffix(filename, ".dart")
	return strings.HasSuffix(name, "_"+toSnakeSuffix(suffix))
}

func parseDartImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "import") {
			continue
		}
		imp := extractQuoted(line)
		if imp == "" || strings.HasPrefix(imp, "dart:") {
			continue
		}
		imports = append(imports, imp)
	}
	return imports
}

func detectFlutterStructure(featureDir string) string {
	subs, err := os.ReadDir(featureDir)
	if err != nil {
		return "flat"
	}
	cleanArch := map[string]bool{"presentation": false, "domain": false, "data": false}
	grouped := map[string]bool{"bloc": false, "screen": false, "repository": false}
	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		n := sub.Name()
		if _, ok := cleanArch[n]; ok {
			cleanArch[n] = true
		}
		if _, ok := grouped[n]; ok {
			grouped[n] = true
		}
	}
	count := 0
	for _, v := range cleanArch {
		if v {
			count++
		}
	}
	if count >= 2 {
		return "clean_architecture"
	}
	if grouped["bloc"] || grouped["screen"] || grouped["repository"] {
		return "grouped"
	}
	return "flat"
}

func deriveFlutterPatterns(imports []string) []string {
	patternMap := map[string]string{
		"package:flutter_bloc/flutter_bloc.dart": "uses flutter_bloc",
		"package:go_router/go_router.dart":        "uses go_router",
		"package:riverpod/riverpod.dart":          "uses riverpod",
		"package:provider/provider.dart":          "uses provider",
		"package:dio/dio.dart":                    "uses dio",
		"package:http/http.dart":                  "uses http",
		"package:equatable/equatable.dart":        "uses equatable",
		"package:get_it/get_it.dart":              "uses get_it",
	}
	seen := map[string]bool{}
	var patterns []string
	for _, imp := range imports {
		if p, ok := patternMap[imp]; ok && !seen[p] {
			patterns = append(patterns, p)
			seen[p] = true
		}
	}
	sort.Strings(patterns)
	return patterns
}

func hasFlutterTestFiles(rootDir, featureName string) bool {
	testCandidates := []string{
		filepath.Join(rootDir, "test", "features", featureName),
		filepath.Join(rootDir, "test", featureName),
	}
	for _, p := range testCandidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return true
		}
	}
	testDir := filepath.Join(rootDir, "test")
	found := false
	_ = filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.Contains(path, featureName) && strings.HasSuffix(path, "_test.dart") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// ── Go ────────────────────────────────────────────────────────────────────────

func buildGoExample(rootDir, featureDir, name string, conv *models.Convention) models.Example {
	relPath, _ := filepath.Rel(rootDir, featureDir)

	files := map[string]string{}
	var allImports []string

	_ = filepath.Walk(featureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if !strings.HasSuffix(base, ".go") || strings.HasSuffix(base, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)

		switch {
		case hasGoSuffix(base, conv.Naming.HandlerSuffix):
			files["handler"] = rel
		case hasGoSuffix(base, conv.Naming.ServiceSuffix):
			files["service"] = rel
		case hasGoSuffix(base, conv.Naming.RepositorySuffix):
			if _, exists := files["repository"]; !exists {
				files["repository"] = rel
			} else {
				files["repository_impl"] = rel
			}
		}

		allImports = append(allImports, parseGoImportsFromFile(path)...)
		return nil
	})

	uniqueImports := dedup(allImports)
	structure := detectGoStructure(featureDir)
	patterns := deriveGoPatterns(uniqueImports)

	meta := models.FeatureMetadata{
		HasHandler:    files["handler"] != "",
		HasService:    files["service"] != "",
		HasRepository: files["repository"] != "",
		HasTest:       hasGoTestFiles(featureDir),
		Imports:       uniqueImports,
		Structure:     structure,
	}

	return models.Example{
		Name:     name,
		Path:     relPath,
		Files:    files,
		Patterns: patterns,
		Metadata: meta,
	}
}

func hasGoSuffix(filename, suffix string) bool {
	if suffix == "" {
		return false
	}
	name := strings.TrimSuffix(filename, ".go")
	return strings.HasSuffix(name, "_"+toSnakeSuffix(suffix))
}

func parseGoImportsFromFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	inBlock := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "import (" {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if inBlock || strings.HasPrefix(line, `import "`) {
			imp := extractQuoted(line)
			if imp != "" && strings.Contains(imp, ".") { // skip stdlib (no dot)
				imports = append(imports, imp)
			}
		}
	}
	return imports
}

func detectGoStructure(featureDir string) string {
	subs, err := os.ReadDir(featureDir)
	if err != nil {
		return "flat"
	}
	for _, sub := range subs {
		if !sub.IsDir() {
			continue
		}
		switch sub.Name() {
		case "delivery", "usecase", "domain", "repository":
			return "clean_architecture"
		}
	}
	return "flat"
}

func deriveGoPatterns(imports []string) []string {
	// prefix → pattern label (versioned imports like chi/v5 matched by prefix)
	patternMap := map[string]string{
		"github.com/gin-gonic/gin":    "uses gin",
		"github.com/go-chi/chi":       "uses chi",
		"github.com/labstack/echo":    "uses echo",
		"github.com/gofiber/fiber":    "uses fiber",
		"github.com/gorilla/mux":      "uses gorilla_mux",
		"gorm.io/gorm":                "uses gorm",
		"github.com/jmoiron/sqlx":     "uses sqlx",
		"go.mongodb.org/mongo-driver": "uses mongodb",
	}
	seen := map[string]bool{}
	var patterns []string
	for _, imp := range imports {
		for prefix, label := range patternMap {
			if (imp == prefix || strings.HasPrefix(imp, prefix+"/")) && !seen[label] {
				patterns = append(patterns, label)
				seen[label] = true
			}
		}
	}
	sort.Strings(patterns)
	return patterns
}

func hasGoTestFiles(featureDir string) bool {
	found := false
	_ = filepath.Walk(featureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// ── shared helpers ────────────────────────────────────────────────────────────

// toSnakeSuffix converts a PascalCase suffix to snake_case (e.g. "UseCase" → "use_case").
func toSnakeSuffix(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(r | 0x20)
	}
	return b.String()
}

func extractQuoted(line string) string {
	for _, q := range []byte{'"', '\''} {
		start := strings.IndexByte(line, q)
		if start == -1 {
			continue
		}
		end := strings.IndexByte(line[start+1:], q)
		if end == -1 {
			continue
		}
		return line[start+1 : start+1+end]
	}
	return ""
}

func dedup(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, v := range items {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
