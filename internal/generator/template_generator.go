package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	repotmpl "github.com/ChonlakanSutthimatmongkhol/repox/templates"
)

// TemplateContext holds values passed to every scaffold template.
type TemplateContext struct {
	FeatureName      string
	PascalName       string
	CamelName        string
	SnakeName        string
	ScreenSuffix     string
	BlocSuffix       string
	EventSuffix      string
	StateSuffix      string
	RepositorySuffix string
	UsecaseSuffix    string
	CommonImports    []string
}

// GeneratedFile pairs an output path with its rendered content.
type GeneratedFile struct {
	Path    string
	Content string
}

// TemplateGenerator renders scaffold templates for a given feature.
type TemplateGenerator struct {
	fs fs.FS
}

// NewTemplateGenerator returns a generator backed by the embedded template FS.
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{fs: repotmpl.FS}
}

// Generate renders all templates in templateName for the given feature and conventions.
func (g *TemplateGenerator) Generate(featureName, templateName string, conv *models.Convention) ([]GeneratedFile, error) {
	ctx := buildContext(featureName, conv)

	pattern := filepath.Join(templateName, "*.tmpl")
	entries, err := fs.Glob(g.fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("generator: glob %s: %w", pattern, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("generator: no templates found for %q", templateName)
	}

	var out []GeneratedFile
	for _, entry := range entries {
		content, err := g.renderFile(entry, ctx)
		if err != nil {
			return nil, err
		}
		outPath := outputPath(entry, ctx, conv)
		out = append(out, GeneratedFile{Path: outPath, Content: content})
	}
	return out, nil
}

func (g *TemplateGenerator) renderFile(tmplPath string, ctx TemplateContext) (string, error) {
	data, err := fs.ReadFile(g.fs, tmplPath)
	if err != nil {
		return "", fmt.Errorf("generator: read template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("generator: parse template %s: %w", tmplPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("generator: render template %s: %w", tmplPath, err)
	}
	return buf.String(), nil
}

// outputPath derives the destination file path from the template filename.
// Template names follow the pattern: <kind>.dart.tmpl → <snakeName>_<kind>.dart
// Uses pattern mappings from config to route files to appropriate subdirectories.
func outputPath(tmplPath string, ctx TemplateContext, conv *models.Convention) string {
	base := filepath.Base(tmplPath)
	// strip .tmpl extension
	name := strings.TrimSuffix(base, ".tmpl")
	// prefix with snake feature name
	outName := ctx.SnakeName + "_" + name

	// bloc_test.dart → test root
	if strings.Contains(name, "test") {
		return filepath.Join(conv.TestRoot, ctx.SnakeName, outName)
	}

	// Check if pattern mappings exist in config
	if len(conv.FeaturesAnalysis.PatternMappings) > 0 {
		// Get mappings for the detected/recommended pattern
		pattern := conv.FeatureStructure
		if pattern == "" {
			pattern = conv.FeaturesAnalysis.RecommendedPattern
		}

		if mappings, ok := conv.FeaturesAnalysis.PatternMappings[pattern]; ok {
			for _, m := range mappings {
				if m.FileName == name {
					if m.Subdir != "" {
						return filepath.Join(conv.FeatureRoot, ctx.SnakeName, m.Subdir, outName)
					}
				}
			}
		}
	}

	// Default: no subdirectory
	return filepath.Join(conv.FeatureRoot, ctx.SnakeName, outName)
}

func buildContext(featureName string, conv *models.Convention) TemplateContext {
	return TemplateContext{
		FeatureName:      featureName,
		PascalName:       ToPascalCase(featureName),
		CamelName:        ToCamelCase(featureName),
		SnakeName:        ToSnakeCase(featureName),
		ScreenSuffix:     conv.Naming.ScreenSuffix,
		BlocSuffix:       conv.Naming.BlocSuffix,
		EventSuffix:      conv.Naming.EventSuffix,
		StateSuffix:      conv.Naming.StateSuffix,
		RepositorySuffix: conv.Naming.RepositorySuffix,
		UsecaseSuffix:    conv.Naming.UsecaseSuffix,
		CommonImports:    conv.CommonImports,
	}
}
