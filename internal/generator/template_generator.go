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

// toGoPackageName converts a feature name to a valid Go package name
// (lowercase, no underscores or hyphens).
func toGoPackageName(featureName string) string {
	s := ToSnakeCase(featureName)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.ToLower(s)
}

// TemplateContext holds values passed to every scaffold template.
type TemplateContext struct {
	FeatureName      string
	FeaturePath      string
	PascalName       string
	CamelName        string
	SnakeName        string
	PackageName      string // Go: lowercase no-separator package name
	ModulePath       string // Go: module path from go.mod
	ScreenSuffix     string
	BlocSuffix       string
	EventSuffix      string
	StateSuffix      string
	RepositorySuffix string
	UsecaseSuffix    string
	HandlerSuffix    string // Go
	ServiceSuffix    string // Go
	CommonImports    []string
	BlocImport       string
	RepositoryImport string
	RequestImport    string
	ResponseImport   string
}

// GeneratedFile pairs an output path with its rendered content.
type GeneratedFile struct {
	Path    string
	Content string
}

// GenerateOptions controls which template roles are rendered.
type GenerateOptions struct {
	Roles []string
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
	return g.GenerateWithOptions(featureName, templateName, conv, GenerateOptions{})
}

// GenerateWithOptions renders selected templates in templateName for the given feature and conventions.
func (g *TemplateGenerator) GenerateWithOptions(featureName, templateName string, conv *models.Convention, opts GenerateOptions) ([]GeneratedFile, error) {
	ctx := buildContext(featureName, conv)

	pattern := filepath.Join(templateName, "*.tmpl")
	entries, err := fs.Glob(g.fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("generator: glob %s: %w", pattern, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("generator: no templates found for %q", templateName)
	}

	allowedRoles := roleSet(opts.Roles)
	var out []GeneratedFile
	for _, entry := range entries {
		kind := templateKind(entry)
		if len(allowedRoles) > 0 && !allowedRoles[kind] {
			continue
		}
		outPath := outputPath(entry, ctx, conv)
		fileCtx := ctx.withImportsFor(outPath, conv)
		content, err := g.renderFile(entry, fileCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, GeneratedFile{Path: outPath, Content: content})
	}
	return out, nil
}

func templateKind(tmplPath string) string {
	base := filepath.Base(tmplPath)
	name := strings.TrimSuffix(base, ".tmpl")
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func roleSet(roles []string) map[string]bool {
	if len(roles) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed[role] = true
	}
	return allowed
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
// Template names follow the pattern: <kind>.<ext>.tmpl → <snakeName>_<kind>.<ext>
func outputPath(tmplPath string, ctx TemplateContext, conv *models.Convention) string {
	base := filepath.Base(tmplPath)
	// strip .tmpl extension
	name := strings.TrimSuffix(base, ".tmpl")
	kind := strings.TrimSuffix(name, filepath.Ext(name))
	// prefix with snake feature name
	outName := ctx.SnakeName + "_" + name

	// *_test.* → test root; for Go, TestRoot == FeatureRoot so tests stay alongside source.
	if strings.Contains(kind, "test") {
		return filepath.Join(conv.TestRoot, ctx.FeaturePath, outName)
	}
	return filepath.Join(conv.FeatureRoot, ctx.FeaturePath, routeForKind(conv, ctx.FeaturePath, kind), outName)
}

func buildContext(featureName string, conv *models.Convention) TemplateContext {
	featurePath := normalizeFeaturePath(featureName)
	leafName := filepath.Base(featurePath)
	return TemplateContext{
		FeatureName:      featureName,
		FeaturePath:      featurePath,
		PascalName:       ToPascalCase(leafName),
		CamelName:        ToCamelCase(leafName),
		SnakeName:        ToSnakeCase(leafName),
		PackageName:      toGoPackageName(leafName),
		ModulePath:       conv.ModulePath,
		ScreenSuffix:     conv.Naming.ScreenSuffix,
		BlocSuffix:       conv.Naming.BlocSuffix,
		EventSuffix:      conv.Naming.EventSuffix,
		StateSuffix:      conv.Naming.StateSuffix,
		RepositorySuffix: conv.Naming.RepositorySuffix,
		UsecaseSuffix:    conv.Naming.UsecaseSuffix,
		HandlerSuffix:    conv.Naming.HandlerSuffix,
		ServiceSuffix:    conv.Naming.ServiceSuffix,
		CommonImports:    conv.CommonImports,
	}
}

func (ctx TemplateContext) withImportsFor(outPath string, conv *models.Convention) TemplateContext {
	ctx.BlocImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "bloc"))
	ctx.RepositoryImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "repository"))
	ctx.RequestImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "request"))
	ctx.ResponseImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "response"))
	return ctx
}

func featureFilePath(conv *models.Convention, ctx TemplateContext, kind string) string {
	outName := ctx.SnakeName + "_" + kind + ".dart"
	return filepath.Join(conv.FeatureRoot, ctx.FeaturePath, routeForKind(conv, ctx.FeaturePath, kind), outName)
}

func normalizeFeaturePath(featureName string) string {
	parts := strings.FieldsFunc(featureName, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return ToSnakeCase(featureName)
	}

	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		normalized = append(normalized, ToSnakeCase(part))
	}
	if len(normalized) == 0 {
		return ToSnakeCase(featureName)
	}
	return filepath.Join(normalized...)
}

func routeForKind(conv *models.Convention, featurePath, kind string) string {
	pattern := conv.FeatureStructure
	if feature, ok := findFeatureConvention(conv, featurePath); ok {
		if feature.FileRoutes != nil {
			if route, exists := feature.FileRoutes[kind]; exists {
				return route
			}
		}
		if feature.Structure != "" {
			pattern = feature.Structure
		}
	}
	if pattern == "" {
		pattern = "flat"
	}
	if conv.PatternMappings != nil {
		if mapping, ok := conv.PatternMappings[pattern]; ok && mapping.FileRoutes != nil {
			return mapping.FileRoutes[kind]
		}
	}
	return defaultRouteForKind(pattern, kind)
}

func findFeatureConvention(conv *models.Convention, featurePath string) (models.FeatureAnalysis, bool) {
	target := filepath.ToSlash(featurePath)
	targetParent := filepath.ToSlash(filepath.Dir(target))
	if targetParent == "." {
		targetParent = ""
	}

	var sibling models.FeatureAnalysis
	hasSibling := false
	for _, feature := range conv.FeaturesAnalysis.Features {
		relPath := featurePathWithoutRoot(conv.FeatureRoot, feature.Path)
		if relPath == target {
			return feature, true
		}
		if filepath.ToSlash(feature.Parent) == targetParent {
			if !hasSibling || feature.Path > sibling.Path {
				sibling = feature
				hasSibling = true
			}
		}
	}
	return sibling, hasSibling
}

func featurePathWithoutRoot(featureRoot, path string) string {
	featureRoot = filepath.ToSlash(strings.Trim(featureRoot, "/"))
	path = filepath.ToSlash(strings.Trim(path, "/"))
	if featureRoot != "" && strings.HasPrefix(path, featureRoot+"/") {
		return strings.TrimPrefix(path, featureRoot+"/")
	}
	return path
}

func defaultRouteForKind(pattern, kind string) string {
	routes := map[string]map[string]string{
		"flat": {
			"bloc": "", "event": "", "state": "", "screen": "", "repository": "",
			"repository_impl": "", "request": "", "response": "", "usecase": "",
		},
		"grouped": {
			"bloc": "bloc", "event": "bloc", "state": "bloc", "screen": "screen",
			"repository": "repository", "repository_impl": "repository",
			"request": "models", "response": "models", "usecase": "usecase",
		},
		"clean_architecture": {
			"bloc": "presentation/bloc", "event": "presentation/bloc",
			"state": "presentation/bloc", "screen": "presentation/screen",
			"repository": "domain/repositories", "repository_impl": "data/repositories",
			"request": "data/models", "response": "data/models",
			"usecase": "domain/usecases",
		},
	}
	if byKind, ok := routes[pattern]; ok {
		return byKind[kind]
	}
	return ""
}

func relativeDartImport(fromPath, toPath string) string {
	rel, err := filepath.Rel(filepath.Dir(fromPath), toPath)
	if err != nil {
		return filepath.ToSlash(toPath)
	}
	return filepath.ToSlash(rel)
}
