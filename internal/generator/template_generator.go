package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/conventions"
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
	FeatureName   string
	FeaturePath   string
	PascalName    string
	CamelName     string
	SnakeName     string
	PackageName   string // Go: lowercase no-separator package name
	ModulePath    string // Go: module path from go.mod
	CommonImports []string
	// Suffixes holds PascalCase class-name suffixes keyed by role name (e.g. "bloc" → "Bloc", "service" → "Service").
	// Populated dynamically from the scanned SuffixRoles convention — no hardcoded role names.
	Suffixes map[string]string
	// Imports holds relative dart import paths keyed by role name (e.g. "bloc", "usecase").
	// Populated dynamically from the active roles being generated — no hardcoded role names.
	Imports   map[string]string
	FileNames map[string]string
	Roles     map[string]TemplateRoleContext
}

// TemplateRoleContext contains scanned anatomy projected into template-friendly fields.
type TemplateRoleContext struct {
	Role           string
	BaseClass      string
	StateBaseClass string
	BaseImports    []string
	AbstractStubs  []string
}

// GeneratedFile pairs an output path with its rendered content.
type GeneratedFile struct {
	Path    string
	Content string
}

// GenerateOptions controls which template roles are rendered.
type GenerateOptions struct {
	Roles         []string
	RolesExplicit bool
	LikeFeature   *models.FeatureAnalysis
	BaseDir       string
}

// TemplateGenerator renders scaffold templates for a given feature.
type TemplateGenerator struct {
	fs fs.FS
}

// NewTemplateGenerator returns a generator backed by the embedded template FS.
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{fs: repotmpl.FS}
}

// TemplateRoles returns the template kinds available for templateName.
func TemplateRoles(templateName string) ([]string, error) {
	pattern := filepath.Join(templateName, "*.tmpl")
	entries, err := fs.Glob(repotmpl.FS, pattern)
	if err != nil {
		return nil, fmt.Errorf("generator: glob %s: %w", pattern, err)
	}
	roles := make([]string, 0, len(entries))
	for _, entry := range entries {
		roles = append(roles, templateKind(entry))
	}
	sort.Strings(roles)
	return roles, nil
}

// Generate renders all templates in templateName for the given feature and conventions.
func (g *TemplateGenerator) Generate(featureName, templateName string, conv *models.Convention) ([]GeneratedFile, error) {
	return g.GenerateWithOptions(featureName, templateName, conv, GenerateOptions{})
}

// GenerateWithOptions renders selected templates in templateName for the given feature and conventions.
func (g *TemplateGenerator) GenerateWithOptions(featureName, templateName string, conv *models.Convention, opts GenerateOptions) ([]GeneratedFile, error) {
	profile := conventions.NewProjectProfile(conv)
	ctx := buildContext(featureName, conv, profile)
	ctx = ctx.withRoleAnatomyFrom(opts.LikeFeature)
	conv = conventionWithLikeTarget(conv, ctx, opts)
	profile = conventions.NewProjectProfile(conv)
	target := targetFeatureFromContext(ctx)

	pattern := filepath.Join(templateName, "*.tmpl")
	entries, err := fs.Glob(g.fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("generator: glob %s: %w", pattern, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("generator: no templates found for %q", templateName)
	}

	allowedRoles := roleSet(opts.Roles)

	// For Go projects with --like, let copy-rename take precedence over templates for
	// roles the like feature already has. This preserves the project's file-naming
	// convention (e.g. "repository.go" instead of the template's "payment_repository.go").
	likeRolesHandledByCopy := map[string]bool{}
	if opts.LikeFeature != nil && opts.BaseDir != "" && conv.ProjectType == "go" {
		for role := range profile.FeatureFiles(*opts.LikeFeature, opts.BaseDir) {
			likeRolesHandledByCopy[role] = true
		}
	}

	// Pre-compute which roles will be rendered so templates can conditionally
	// inject dependencies (e.g. usecase) only when those roles are actually generated.
	activeRoles := map[string]bool{}
	for _, entry := range entries {
		kind := templateKind(entry)
		if likeRolesHandledByCopy[kind] {
			continue
		}
		if len(allowedRoles) == 0 || allowedRoles[kind] {
			activeRoles[kind] = true
		}
	}

	generatedRoles := map[string]bool{}
	var out []GeneratedFile
	for _, entry := range entries {
		kind := templateKind(entry)
		if len(allowedRoles) > 0 && !allowedRoles[kind] {
			continue
		}
		// Defer to copy-rename for roles the like feature already provides.
		if likeRolesHandledByCopy[kind] {
			continue
		}
		outPath := profile.OutputPath(entry, target)
		fileCtx := ctx.withImportsFor(outPath, profile, activeRoles)
		content, err := g.renderFile(entry, fileCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, GeneratedFile{Path: outPath, Content: content})
		generatedRoles[kind] = true
	}
	out = append(out, renderLikeAncillaryFiles(ctx, profile, opts, allowedRoles, generatedRoles)...)
	return out, nil
}

func conventionWithLikeTarget(conv *models.Convention, ctx TemplateContext, opts GenerateOptions) *models.Convention {
	if opts.LikeFeature == nil {
		return conv
	}
	convCopy := *conv
	analysis := conv.FeaturesAnalysis
	liked := *opts.LikeFeature
	liked.Path = filepath.ToSlash(filepath.Join(conv.FeatureRoot, ctx.FeaturePath))
	liked.Parent = filepath.ToSlash(filepath.Dir(ctx.FeaturePath))
	if liked.Parent == "." {
		liked.Parent = ""
	}
	liked.Name = filepath.Base(ctx.FeaturePath)
	analysis.Features = append(append([]models.FeatureAnalysis{}, analysis.Features...), liked)
	convCopy.FeaturesAnalysis = analysis
	if liked.Structure != "" {
		convCopy.FeatureStructure = liked.Structure
	}
	return &convCopy
}

func renderLikeAncillaryFiles(ctx TemplateContext, profile *conventions.ProjectProfile, opts GenerateOptions, allowedRoles map[string]bool, generatedRoles map[string]bool) []GeneratedFile {
	if opts.LikeFeature == nil || strings.TrimSpace(opts.BaseDir) == "" {
		return nil
	}
	plan := profile.RenamePlan(*opts.LikeFeature, targetFeatureFromContext(ctx), opts.BaseDir)
	files := plan.Files
	roles := make([]string, 0, len(files))
	for role := range files {
		roles = append(roles, role)
	}
	sort.Strings(roles)

	var out []GeneratedFile
	for _, role := range roles {
		if generatedRoles[role] {
			continue
		}
		if opts.RolesExplicit && len(allowedRoles) > 0 && !allowedRoles[role] {
			continue
		}
		outPath := plan.RewritePath(files[role], opts.BaseDir)
		content, ok := renderLikeSourcePath(files[role], plan, opts)
		if !ok || outPath == "" {
			continue
		}
		out = append(out, GeneratedFile{Path: outPath, Content: content})
		generatedRoles[role] = true
	}
	return out
}

func renderLikeSourcePath(sourcePath string, plan conventions.FeatureRenamePlan, opts GenerateOptions) (string, bool) {
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(opts.BaseDir, sourcePath)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", false
	}
	content := stripCrossFeatureImports(string(data), opts.LikeFeature)
	return rewriteLikeFeatureText(content, plan, *opts.LikeFeature, opts.BaseDir), true
}

// stripCrossFeatureImports removes Dart import lines that reference other features
// (not the source feature itself, common/, or core/ packages), preventing cross-feature
// business logic from leaking into freshly scaffolded features.
func stripCrossFeatureImports(content string, like *models.FeatureAnalysis) string {
	if like == nil {
		return content
	}
	sourcePath := filepath.ToSlash(like.Path)
	featureRoot := ""
	if idx := strings.Index(sourcePath, "features/"); idx >= 0 {
		featureRoot = strings.TrimPrefix(sourcePath[:idx+len("features")], "lib/")
	}
	if featureRoot == "" {
		return content
	}
	// Source feature path relative to the lib root for import matching
	// e.g. "features/investment/fund_list"
	sourceInImport := strings.TrimPrefix(sourcePath, "lib/")

	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") && strings.HasSuffix(trimmed, ";") &&
			strings.Contains(trimmed, featureRoot+"/") &&
			!strings.Contains(trimmed, sourceInImport) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func rewriteLikeFeatureText(content string, plan conventions.FeatureRenamePlan, like models.FeatureAnalysis, baseDir string) string {
	replacements := plan.Replacements()
	ownedIdentifiers := likeOwnedIdentifiers(like, plan, baseDir, content)
	for source, target := range likeIdentifierReplacements(ownedIdentifiers, plan.SourcePascal, plan.SourceCamel, plan.SourceSnake, plan.TargetPascal, plan.TargetCamel, plan.TargetSnake) {
		replacements[source] = target
	}

	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		if key != "" && key != replacements[key] {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	for _, key := range keys {
		content = strings.ReplaceAll(content, key, replacements[key])
	}
	return content
}

func likeOwnedIdentifiers(like models.FeatureAnalysis, plan conventions.FeatureRenamePlan, baseDir, content string) map[string]bool {
	owned := map[string]bool{}
	addLeadingFeatureIdentifiers(owned, content, plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
	addOwnedIdentifiers(owned, declaredIdentifiers(content), plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
	for _, anatomy := range like.Anatomy {
		addOwnedIdentifiers(owned, anatomy.ClassNames, plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
		addOwnedIdentifiers(owned, anatomy.Methods, plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
	}
	addOwnedIdentifiers(owned, fileBaseIdentifiers(plan), plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
	if strings.TrimSpace(baseDir) == "" {
		return owned
	}
	for _, path := range plan.Files {
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		addOwnedIdentifiers(owned, declaredIdentifiers(string(data)), plan.SourcePascal, plan.SourceCamel, plan.SourceSnake)
	}
	return owned
}

func addLeadingFeatureIdentifiers(owned map[string]bool, content, sourcePascal, sourceCamel, sourceSnake string) {
	for _, token := range identifierTokens(content) {
		if token == sourceSnake {
			continue
		}
		if strings.HasPrefix(token, sourcePascal) ||
			strings.HasPrefix(token, sourceCamel) ||
			strings.HasPrefix(token, sourceSnake+"_") {
			owned[token] = true
		}
	}
}

func addOwnedIdentifiers(owned map[string]bool, items []string, sourcePascal, sourceCamel, sourceSnake string) {
	for _, item := range items {
		if shouldConsiderFeatureIdentifier(item, sourcePascal, sourceCamel, sourceSnake) {
			owned[item] = true
		}
	}
}

func shouldConsiderFeatureIdentifier(item, sourcePascal, sourceCamel, sourceSnake string) bool {
	return item != "" && (strings.Contains(item, sourcePascal) || strings.Contains(item, sourceCamel) || strings.Contains(item, sourceSnake))
}

func fileBaseIdentifiers(plan conventions.FeatureRenamePlan) []string {
	var identifiers []string
	for _, path := range plan.Files {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if base == "" {
			continue
		}
		identifiers = append(identifiers, base, ToPascalCase(base), ToCamelCase(base))
	}
	return identifiers
}

func declaredIdentifiers(content string) []string {
	tokens := orderedIdentifierTokens(content)
	var out []string
	for i, token := range tokens {
		if i+1 >= len(tokens) {
			continue
		}
		switch token {
		case "class", "enum", "extension", "mixin", "type", "interface", "struct":
			out = append(out, tokens[i+1])
		}
	}
	return out
}

func likeIdentifierReplacements(ownedIdentifiers map[string]bool, sourcePascal, sourceCamel, sourceSnake, targetPascal, targetCamel, targetSnake string) map[string]string {
	replacements := map[string]string{}
	for match := range ownedIdentifiers {
		if !strings.Contains(match, sourcePascal) && !strings.Contains(match, sourceCamel) && !strings.Contains(match, sourceSnake) {
			continue
		}
		if match == sourceSnake {
			continue
		}
		next := strings.ReplaceAll(match, sourcePascal, targetPascal)
		next = strings.ReplaceAll(next, sourceCamel, targetCamel)
		next = strings.ReplaceAll(next, sourceSnake, targetSnake)
		if next != match {
			replacements[match] = next
		}
	}
	return replacements
}

func identifierTokens(content string) []string {
	seen := map[string]bool{}
	var tokens []string
	for _, token := range orderedIdentifierTokens(content) {
		if !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func orderedIdentifierTokens(content string) []string {
	var tokens []string
	for i := 0; i < len(content); {
		if !isIdentifierStart(content[i]) {
			i++
			continue
		}
		start := i
		i++
		for i < len(content) && isIdentifierChar(content[i]) {
			i++
		}
		tokens = append(tokens, content[start:i])
	}
	return tokens
}

func isIdentifierStart(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isIdentifierChar(ch byte) bool {
	return isIdentifierStart(ch) || (ch >= '0' && ch <= '9')
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

func buildContext(featureName string, conv *models.Convention, profile *conventions.ProjectProfile) TemplateContext {
	featurePath := normalizeFeaturePath(featureName)
	leafName := filepath.Base(featurePath)
	return TemplateContext{
		FeatureName:   featureName,
		FeaturePath:   featurePath,
		PascalName:    ToPascalCase(leafName),
		CamelName:     ToCamelCase(leafName),
		SnakeName:     ToSnakeCase(leafName),
		PackageName:   toGoPackageName(leafName),
		ModulePath:    conv.ModulePath,
		Suffixes:      profile.Suffixes(),
		CommonImports: conv.CommonImports,
	}
}

func (ctx TemplateContext) withRoleAnatomyFrom(like *models.FeatureAnalysis) TemplateContext {
	ctx.Roles = map[string]TemplateRoleContext{}
	if like == nil {
		return ctx
	}
	for role, anatomy := range like.Anatomy {
		roleCtx := TemplateRoleContext{
			Role:          role,
			BaseImports:   append(baseClassImports(anatomy.Imports, anatomy.BaseClasses, anatomy.Mixins), abstractStubImports(anatomy.AbstractOverrides, anatomy.Imports)...),
			AbstractStubs: buildAbstractStubs(anatomy.AbstractOverrides),
		}
		for _, base := range anatomy.BaseClasses {
			if strings.HasSuffix(base, "State") {
				roleCtx.StateBaseClass = base
				continue
			}
			if roleCtx.BaseClass == "" {
				roleCtx.BaseClass = base
			}
		}
		if roleCtx.BaseClass == "" && len(anatomy.BaseClasses) > 0 {
			roleCtx.BaseClass = anatomy.BaseClasses[0]
		}
		ctx.Roles[role] = roleCtx
	}
	return ctx
}

// baseClassImports filters imports to those that likely provide the given base classes.
// It keeps imports from "core/base" paths and any import whose filename matches a base class name.
// baseClassImports finds imports that provide the base classes and their direct
// dependencies — derived entirely from the scanned anatomy with no path hardcoding.
//
// Strategy:
//  1. Match imports whose filename (snake_case) overlaps with any base class or mixin name.
//  2. Expand: include all other anatomy imports that share the same directory as a matched import,
//     because base classes often live alongside their event/state/mixin siblings.
func baseClassImports(imports []string, baseClasses []string, mixins []string) []string {
	allClasses := append(append([]string{}, baseClasses...), mixins...)

	// Pass 1: imports whose filename matches a base class or mixin name.
	matchedDirs := map[string]bool{}
	matched := map[string]bool{}
	for _, imp := range imports {
		impBase := strings.ToLower(strings.TrimSuffix(filepath.Base(imp), ".dart"))
		for _, cls := range allClasses {
			if cls == "" {
				continue
			}
			if strings.Contains(impBase, strings.ToLower(ToSnakeCase(cls))) {
				matched[imp] = true
				matchedDirs[filepath.Dir(imp)] = true
				break
			}
		}
	}

	// Pass 2: include all imports from the same directories as matched ones.
	// This captures sibling files (e.g. base_event.dart next to base_bloc_screen.dart).
	var result []string
	seen := map[string]bool{}
	for _, imp := range imports {
		if matchedDirs[filepath.Dir(imp)] && !seen[imp] {
			result = append(result, imp)
			seen[imp] = true
		}
	}
	return result
}

// buildAbstractStubs turns scanned "@override" method signatures into
// "  @override\n  Sig => throw UnimplementedError();" stub strings.
func buildAbstractStubs(overrides []string) []string {
	stubs := make([]string, 0, len(overrides))
	for _, sig := range overrides {
		stubs = append(stubs, sig+" => throw UnimplementedError();")
	}
	return stubs
}

// abstractStubImports finds imports that provide types referenced in the abstract override
// signatures. It extracts uppercase type names from the signatures and matches them
// against import filenames — generic, no project-specific hardcoding.
func abstractStubImports(overrides []string, allImports []string) []string {
	if len(overrides) == 0 {
		return nil
	}
	// Collect all uppercase type names from the override signatures
	typeNames := map[string]bool{}
	for _, sig := range overrides {
		for _, tok := range strings.FieldsFunc(sig, func(r rune) bool {
			return !isIdentChar(r)
		}) {
			if len(tok) > 0 && tok[0] >= 'A' && tok[0] <= 'Z' {
				typeNames[tok] = true
			}
		}
	}

	seen := map[string]bool{}
	var result []string
	for _, imp := range allImports {
		impBase := strings.ToLower(strings.TrimSuffix(filepath.Base(imp), ".dart"))
		for typ := range typeNames {
			if strings.Contains(impBase, strings.ToLower(ToSnakeCase(typ))) && !seen[imp] {
				result = append(result, imp)
				seen[imp] = true
				break
			}
		}
	}
	return result
}

func isIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func (ctx TemplateContext) withImportsFor(outPath string, profile *conventions.ProjectProfile, activeRoles map[string]bool) TemplateContext {
	ctx.Imports = map[string]string{}
	ctx.FileNames = map[string]string{}
	target := targetFeatureFromContext(ctx)
	for _, role := range profile.RoleNames() {
		ctx.FileNames[role] = profile.FileNameForRole(target, role, ".dart")
	}
	for role := range activeRoles {
		path := profile.ImportFor(outPath, target, role)
		if path != "" {
			ctx.Imports[role] = path
		}
	}
	return ctx
}

func targetFeatureFromContext(ctx TemplateContext) conventions.TargetFeature {
	return conventions.TargetFeature{
		Name:        ctx.FeatureName,
		Path:        filepath.ToSlash(ctx.FeaturePath),
		Leaf:        filepath.Base(ctx.FeaturePath),
		Snake:       ctx.SnakeName,
		Pascal:      ctx.PascalName,
		Camel:       ctx.CamelName,
		PackageName: ctx.PackageName,
	}
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
