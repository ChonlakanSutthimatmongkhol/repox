package scanner

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DetectCommonImports reads all source files in featureRoot and returns the
// top 10 most frequently used imports, excluding standard library imports.
func DetectCommonImports(featureRoot, projectType string) ([]string, error) {
	freq := map[string]int{}

	err := filepath.WalkDir(featureRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch projectType {
		case "flutter", "dart":
			if !strings.HasSuffix(path, ".dart") {
				return nil
			}
			imports, e := parseDartImports(path)
			if e != nil {
				return nil
			}
			for _, imp := range imports {
				freq[imp]++
			}
		case "go":
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			imports, e := parseGoImports(path)
			if e != nil {
				return nil
			}
			for _, imp := range imports {
				freq[imp]++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return topN(freq, 10), nil
}

func parseDartImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "import ") {
			continue
		}
		// Skip dart: core imports
		if strings.Contains(line, "'dart:") || strings.Contains(line, `"dart:`) {
			continue
		}
		// Extract package path from import 'package:...' or import '...'
		imp := extractQuoted(line)
		if imp != "" {
			imports = append(imports, imp)
		}
	}
	return imports, scanner.Err()
}

func parseGoImports(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	inBlock := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == `import (` {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if inBlock || strings.HasPrefix(line, `import "`) {
			imp := extractQuoted(line)
			if imp != "" && !isStdLib(imp) {
				imports = append(imports, imp)
			}
		}
	}
	return imports, scanner.Err()
}

func extractQuoted(s string) string {
	for _, q := range []string{`"`, `'`} {
		start := strings.Index(s, q)
		if start < 0 {
			continue
		}
		end := strings.LastIndex(s, q)
		if end > start {
			return s[start+1 : end]
		}
	}
	return ""
}

func isStdLib(imp string) bool {
	return !strings.Contains(imp, ".")
}

func topN(freq map[string]int, n int) []string {
	type kv struct {
		key   string
		count int
	}
	pairs := make([]kv, 0, len(freq))
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})
	result := make([]string, 0, n)
	for i, p := range pairs {
		if i >= n {
			break
		}
		result = append(result, p.key)
	}
	return result
}
