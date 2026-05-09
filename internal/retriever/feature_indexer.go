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
	relPath, _ := filepath.Rel(rootDir, featureDir)

	files := map[string]string{}
	var allImports []string

	_ = filepath.Walk(featureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".dart") {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)
		base := filepath.Base(path)

		// Assign role by matching naming convention suffixes.
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

		imports := parseDartImports(path)
		allImports = append(allImports, imports...)
		return nil
	})

	// Deduplicate imports.
	importSet := map[string]bool{}
	for _, imp := range allImports {
		importSet[imp] = true
	}
	uniqueImports := make([]string, 0, len(importSet))
	for imp := range importSet {
		uniqueImports = append(uniqueImports, imp)
	}
	sort.Strings(uniqueImports)

	structure := detectStructure(featureDir)
	patterns := derivePatterns(uniqueImports)

	meta := models.FeatureMetadata{
		HasBloc:       files["bloc"] != "",
		HasScreen:     files["screen"] != "",
		HasRepository: files["repository"] != "",
		HasUseCase:    files["usecase"] != "",
		HasTest:       hasTestFiles(rootDir, name),
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
	snakeSuffix := toSnakeSuffix(suffix)
	return strings.HasSuffix(name, "_"+snakeSuffix)
}

// toSnakeSuffix converts a PascalCase suffix to snake_case (e.g. "UseCase" → "use_case").
func toSnakeSuffix(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(r | 0x20) // to lower
	}
	return b.String()
}

func parseDartImports(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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

func detectStructure(featureDir string) string {
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

func derivePatterns(imports []string) []string {
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

func hasTestFiles(rootDir, featureName string) bool {
	testCandidates := []string{
		filepath.Join(rootDir, "test", "features", featureName),
		filepath.Join(rootDir, "test", featureName),
	}
	for _, p := range testCandidates {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return true
		}
	}
	// Also check for any *_test.dart under test/ with featureName in path.
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
