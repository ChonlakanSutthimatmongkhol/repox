// Package templatex extracts simple Repox templates from existing feature files.
package templatex

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/mapgen"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ExtractOptions describes a template extraction request.
type ExtractOptions struct {
	Name string
	From string
	Root string
}

// ExtractResult lists generated template files.
type ExtractResult struct {
	Dir   string
	Files []string
}

// Extract creates a first-pass template from an indexed feature.
func Extract(conv models.Convention, opts ExtractOptions) (ExtractResult, error) {
	if strings.TrimSpace(opts.Name) == "" {
		return ExtractResult{}, fmt.Errorf("template: --name is required")
	}
	if strings.TrimSpace(opts.From) == "" {
		return ExtractResult{}, fmt.Errorf("template: --from is required")
	}
	root := opts.Root
	if root == "" {
		root = "."
	}
	feature, ok := mapgen.FindFeature(conv, opts.From)
	if !ok {
		return ExtractResult{}, fmt.Errorf("template: source feature %q not found in scanned conventions", opts.From)
	}
	templateDir := config.RepoxPath(filepath.Join("templates", opts.Name))
	filesDir := filepath.Join(templateDir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return ExtractResult{}, err
	}

	roles := make([]string, 0, len(feature.Files))
	for role := range feature.Files {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	sourceLeaf := feature.Name
	if sourceLeaf == "" {
		sourceLeaf = filepath.Base(feature.Path)
	}
	module := feature.Parent
	if module == "" {
		rel := strings.TrimPrefix(filepath.ToSlash(feature.Path), strings.Trim(filepath.ToSlash(conv.FeatureRoot), "/")+"/")
		module = filepath.Dir(rel)
		if module == "." {
			module = ""
		}
	}

	var written []string
	for _, role := range roles {
		sourceRel := feature.Files[role]
		if sourceRel == "" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, sourceRel))
		if err != nil {
			continue
		}
		outName := role + filepath.Ext(sourceRel) + ".tmpl"
		outPath := filepath.Join(filesDir, outName)
		content := replaceFeatureTokens(string(data), sourceLeaf, module)
		if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
			return ExtractResult{}, err
		}
		written = append(written, outPath)
	}

	manifest := buildManifest(opts.Name, feature, roles)
	manifestPath := filepath.Join(templateDir, "template.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return ExtractResult{}, err
	}
	written = append([]string{manifestPath}, written...)
	return ExtractResult{Dir: templateDir, Files: written}, nil
}

func replaceFeatureTokens(content, featureName, module string) string {
	snake := generator.ToSnakeCase(featureName)
	pascal := generator.ToPascalCase(featureName)
	kebab := strings.ReplaceAll(snake, "_", "-")
	camel := strings.TrimPrefix(generator.ToPascalCase(featureName), "_")
	if camel != "" {
		camel = strings.ToLower(camel[:1]) + camel[1:]
	}
	replacements := []struct {
		old string
		new string
	}{
		{pascal, "{{FeaturePascal}}"},
		{camel, "{{featureCamel}}"},
		{snake, "{{feature_snake}}"},
		{kebab, "{{feature-kebab}}"},
	}
	if module != "" && module != "." {
		replacements = append(replacements, struct {
			old string
			new string
		}{module, "{{module}}"})
	}
	for _, item := range replacements {
		if item.old != "" {
			content = strings.ReplaceAll(content, item.old, item.new)
		}
	}
	return content
}

func buildManifest(name string, feature models.FeatureAnalysis, roles []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\n", name)
	fmt.Fprintf(&b, "source_feature: %s\n", feature.Path)
	fmt.Fprintln(&b, "variables:")
	fmt.Fprintln(&b, "  - FeaturePascal")
	fmt.Fprintln(&b, "  - featureCamel")
	fmt.Fprintln(&b, "  - feature_snake")
	fmt.Fprintln(&b, "  - feature-kebab")
	fmt.Fprintln(&b, "  - module")
	fmt.Fprintln(&b, "roles:")
	for _, role := range roles {
		fmt.Fprintf(&b, "  - %s\n", role)
	}
	return b.String()
}
