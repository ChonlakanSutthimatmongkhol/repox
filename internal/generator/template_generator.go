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
	UsecaseImport    string
	RepositoryImport string
	RequestImport    string
	ResponseImport   string
	BlocBaseClass     string   // e.g. "BaseBlocScreen" when derived from like anatomy
	BlocBaseImports   []string // imports required by BlocBaseClass
	BlocAbstractStubs []string // "@override Sig => throw UnimplementedError();" stubs
	ScreenBaseClass   string   // e.g. "BaseStatefulWidget" when derived from like anatomy
	ScreenStateBase   string   // e.g. "BaseState" when derived from like anatomy
	ScreenBaseImports []string // imports required by ScreenBaseClass
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
	ctx := buildContext(featureName, conv)
	ctx = ctx.withBaseClassesFrom(opts.LikeFeature)
	conv = conventionWithLikeTarget(conv, ctx, opts)

	pattern := filepath.Join(templateName, "*.tmpl")
	entries, err := fs.Glob(g.fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("generator: glob %s: %w", pattern, err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("generator: no templates found for %q", templateName)
	}

	allowedRoles := roleSet(opts.Roles)

	// Pre-compute which roles will be rendered so templates can conditionally
	// inject dependencies (e.g. usecase) only when those roles are actually generated.
	activeRoles := map[string]bool{}
	for _, entry := range entries {
		kind := templateKind(entry)
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
		outPath := outputPath(entry, ctx, conv)
		fileCtx := ctx.withImportsFor(outPath, conv, activeRoles)
		content, err := g.renderFile(entry, fileCtx)
		if err != nil {
			return nil, err
		}
		out = append(out, GeneratedFile{Path: outPath, Content: content})
		generatedRoles[kind] = true
	}
	out = append(out, renderLikeAncillaryFiles(ctx, conv, opts, allowedRoles, generatedRoles)...)
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

func renderLikeAncillaryFiles(ctx TemplateContext, conv *models.Convention, opts GenerateOptions, allowedRoles map[string]bool, generatedRoles map[string]bool) []GeneratedFile {
	if opts.LikeFeature == nil || strings.TrimSpace(opts.BaseDir) == "" {
		return nil
	}
	files := likeFeatureFiles(*opts.LikeFeature, opts.BaseDir, conv.Naming)
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
		outPath := rewriteLikeFeaturePath(files[role], ctx, conv, *opts.LikeFeature, opts.BaseDir)
		content, ok := renderLikeSourcePath(files[role], ctx, conv, opts)
		if !ok || outPath == "" {
			continue
		}
		out = append(out, GeneratedFile{Path: outPath, Content: content})
		generatedRoles[role] = true
	}
	return out
}

func renderLikeSourcePath(sourcePath string, ctx TemplateContext, conv *models.Convention, opts GenerateOptions) (string, bool) {
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(opts.BaseDir, sourcePath)
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", false
	}
	content := stripCrossFeatureImports(string(data), conv, *opts.LikeFeature)
	return rewriteLikeFeatureText(content, ctx, conv, *opts.LikeFeature, opts.BaseDir), true
}

// stripCrossFeatureImports removes Dart import lines that reference other features
// (not the source feature itself, common/, or core/ packages), preventing cross-feature
// business logic from leaking into freshly scaffolded features.
func stripCrossFeatureImports(content string, conv *models.Convention, like models.FeatureAnalysis) string {
	featureRoot := strings.TrimPrefix(filepath.ToSlash(conv.FeatureRoot), "lib/")
	if featureRoot == "" {
		return content
	}
	sourcePath := filepath.ToSlash(like.Path)
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

func likeFeatureFiles(feature models.FeatureAnalysis, baseDir string, naming ...models.NamingConvention) map[string]string {
	files := map[string]string{}
	for role, path := range feature.Files {
		if strings.TrimSpace(path) != "" {
			files[role] = path
		}
	}
	for role, anatomy := range feature.Anatomy {
		if strings.TrimSpace(anatomy.Path) != "" {
			files[role] = anatomy.Path
		}
	}
	if strings.TrimSpace(baseDir) == "" || strings.TrimSpace(feature.Path) == "" {
		return files
	}
	featureDir := feature.Path
	if !filepath.IsAbs(featureDir) {
		featureDir = filepath.Join(baseDir, featureDir)
	}
	_ = filepath.WalkDir(featureDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != featureDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if isGeneratedLikeSourceFile(path) {
			return nil
		}
		if ext := filepath.Ext(path); ext != ".dart" && ext != ".go" {
			return nil
		}
		n := models.NamingConvention{}
		if len(naming) > 0 {
			n = naming[0]
		}
		role := roleForLikePath(filepath.Base(feature.Path), path, n)
		if role == "" {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if hasLikePath(files, rel) {
			return nil
		}
		role = uniqueRole(role, rel, files)
		files[role] = rel
		return nil
	})
	return files
}

func isGeneratedLikeSourceFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".g.dart") ||
		strings.HasSuffix(base, ".freezed.dart") ||
		strings.HasSuffix(base, ".gen.dart") ||
		strings.HasSuffix(base, ".generated.dart")
}

func hasLikePath(files map[string]string, path string) bool {
	for _, existing := range files {
		if filepath.ToSlash(existing) == path {
			return true
		}
	}
	return false
}

func uniqueRole(role, relPath string, files map[string]string) string {
	if _, exists := files[role]; !exists {
		return role
	}
	dirRole := strings.TrimSuffix(filepath.ToSlash(relPath), filepath.Ext(relPath))
	dirRole = strings.NewReplacer("/", "_", "\\", "_").Replace(dirRole)
	dirRole = strings.Trim(dirRole, "_")
	if dirRole == "" {
		dirRole = role
	}
	candidate := dirRole
	for i := 2; ; i++ {
		if _, exists := files[candidate]; !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", dirRole, i)
	}
}

func roleForLikePath(featureName, path string, naming models.NamingConvention) string {
	filename := filepath.Base(path)
	if role := knownTemplateRole(filename, naming); role != "" {
		return role
	}
	ext := filepath.Ext(filename)
	if ext != ".dart" && ext != ".go" {
		return ""
	}
	base := strings.TrimSuffix(filename, ext)
	featureSnake := ToSnakeCase(featureName)
	if strings.HasPrefix(base, featureSnake+"_") {
		base = strings.TrimPrefix(base, featureSnake+"_")
	}
	return strings.Trim(base, "_")
}

// knownTemplateRole maps a filename to a role using the SuffixRoles map from scan.
// No hardcoded suffix→role list — the scanner builds and stores this in conventions.json.
func knownTemplateRole(filename string, naming models.NamingConvention) string {
	if len(naming.SuffixRoles) == 0 {
		return ""
	}
	base := strings.ToLower(strings.TrimSuffix(filename, filepath.Ext(filename)))
	// Check longer suffixes first to prefer "repositoryimpl" over "repository".
	type entry struct{ suffix, role string }
	entries := make([]entry, 0, len(naming.SuffixRoles))
	for suffix, role := range naming.SuffixRoles {
		entries = append(entries, entry{suffix, role})
	}
	sort.Slice(entries, func(i, j int) bool { return len(entries[i].suffix) > len(entries[j].suffix) })
	for _, e := range entries {
		if strings.HasSuffix(base, "_"+e.suffix) {
			return e.role
		}
	}
	return ""
}

func rewriteLikeFeaturePath(path string, ctx TemplateContext, conv *models.Convention, like models.FeatureAnalysis, baseDir string) string {
	if filepath.IsAbs(path) {
		if rel, err := filepath.Rel(baseDir, path); err == nil {
			path = rel
		}
	}
	return rewriteLikeFeatureText(filepath.ToSlash(path), ctx, conv, like, baseDir)
}

func rewriteLikeFeatureText(content string, ctx TemplateContext, conv *models.Convention, like models.FeatureAnalysis, baseDir string) string {
	sourcePath := featurePathWithoutRoot(conv.FeatureRoot, like.Path)
	sourceLeaf := filepath.Base(sourcePath)
	sourceSnake := ToSnakeCase(sourceLeaf)
	sourcePascal := ToPascalCase(sourceLeaf)
	sourceCamel := ToCamelCase(sourceLeaf)
	targetPath := filepath.ToSlash(ctx.FeaturePath)
	targetSnake := ctx.SnakeName
	targetPascal := ctx.PascalName
	targetCamel := ctx.CamelName

	replacements := map[string]string{
		filepath.ToSlash(sourcePath): targetPath,
		displayName(sourceSnake):     displayName(targetSnake),
	}
	if sourcePath != "" {
		replacements[filepath.ToSlash(filepath.Join(conv.FeatureRoot, sourcePath))] = filepath.ToSlash(filepath.Join(conv.FeatureRoot, targetPath))
	}
	for source, target := range likeFileBaseReplacements(like, baseDir, sourceSnake, targetSnake) {
		replacements[source] = target
	}
	ownedIdentifiers := likeOwnedIdentifiers(like, baseDir, content, sourcePascal, sourceCamel, sourceSnake)
	for source, target := range likeIdentifierReplacements(ownedIdentifiers, sourcePascal, sourceCamel, sourceSnake, targetPascal, targetCamel, targetSnake) {
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

func likeFileBaseReplacements(like models.FeatureAnalysis, baseDir, sourceSnake, targetSnake string) map[string]string {
	replacements := map[string]string{}
	for _, path := range likeFeatureFiles(like, baseDir) {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if !strings.HasPrefix(base, sourceSnake) {
			continue
		}
		replacements[base] = targetSnake + strings.TrimPrefix(base, sourceSnake)
	}
	return replacements
}

func likeOwnedIdentifiers(like models.FeatureAnalysis, baseDir, content, sourcePascal, sourceCamel, sourceSnake string) map[string]bool {
	owned := map[string]bool{}
	addLeadingFeatureIdentifiers(owned, content, sourcePascal, sourceCamel, sourceSnake)
	addOwnedIdentifiers(owned, declaredIdentifiers(content), sourcePascal, sourceCamel, sourceSnake)
	for _, anatomy := range like.Anatomy {
		addOwnedIdentifiers(owned, anatomy.ClassNames, sourcePascal, sourceCamel, sourceSnake)
		addOwnedIdentifiers(owned, anatomy.Methods, sourcePascal, sourceCamel, sourceSnake)
	}
	addOwnedIdentifiers(owned, fileBaseIdentifiers(like, baseDir), sourcePascal, sourceCamel, sourceSnake)
	if strings.TrimSpace(baseDir) == "" {
		return owned
	}
	for _, path := range likeFeatureFiles(like, baseDir) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		addOwnedIdentifiers(owned, declaredIdentifiers(string(data)), sourcePascal, sourceCamel, sourceSnake)
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

func fileBaseIdentifiers(like models.FeatureAnalysis, baseDir string) []string {
	var identifiers []string
	for _, path := range likeFeatureFiles(like, baseDir) {
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

func displayName(snake string) string {
	parts := strings.Split(snake, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
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

func (ctx TemplateContext) withBaseClassesFrom(like *models.FeatureAnalysis) TemplateContext {
	if like == nil {
		return ctx
	}
	if a, ok := like.Anatomy["bloc"]; ok && len(a.BaseClasses) > 0 {
		ctx.BlocBaseClass = a.BaseClasses[0]
		ctx.BlocBaseImports = append(baseClassImports(a.Imports, a.BaseClasses, a.Mixins), abstractStubImports(a.AbstractOverrides, a.Imports)...)
		ctx.BlocAbstractStubs = buildAbstractStubs(a.AbstractOverrides)
	}
	if a, ok := like.Anatomy["screen"]; ok {
		for _, base := range a.BaseClasses {
			switch {
			case strings.HasSuffix(base, "State"):
				ctx.ScreenStateBase = base
			default:
				ctx.ScreenBaseClass = base
			}
		}
		ctx.ScreenBaseImports = baseClassImports(a.Imports, a.BaseClasses, a.Mixins)
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

func (ctx TemplateContext) withImportsFor(outPath string, conv *models.Convention, activeRoles map[string]bool) TemplateContext {
	ctx.BlocImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "bloc"))
	ctx.RepositoryImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "repository"))
	ctx.RequestImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "request"))
	ctx.ResponseImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "response"))
	// Only set UsecaseImport when a usecase file is actually being generated.
	// This lets the bloc template conditionally inject the dependency.
	if activeRoles["usecase"] {
		ctx.UsecaseImport = relativeDartImport(outPath, featureFilePath(conv, ctx, "usecase"))
	}
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
	return ""
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

func relativeDartImport(fromPath, toPath string) string {
	rel, err := filepath.Rel(filepath.Dir(fromPath), toPath)
	if err != nil {
		return filepath.ToSlash(toPath)
	}
	return filepath.ToSlash(rel)
}
