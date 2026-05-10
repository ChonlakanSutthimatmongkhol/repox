package conventions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// TargetFeature is the normalized feature identity used for generation.
type TargetFeature struct {
	Name        string
	Path        string
	Leaf        string
	Snake       string
	Pascal      string
	Camel       string
	PackageName string
}

// ProjectProfile is the single interpretation layer for scanned conventions.
type ProjectProfile struct {
	conv *models.Convention
}

func NewProjectProfile(conv *models.Convention) *ProjectProfile {
	return &ProjectProfile{conv: conv}
}

func (p *ProjectProfile) ValidateScanned() error {
	if p == nil || p.conv == nil {
		return fmt.Errorf("generate: missing conventions; run `repox scan` before generating")
	}
	if p.conv.FeatureRoot == "" || len(p.conv.FeaturesAnalysis.Features) == 0 || len(p.conv.Roles) == 0 {
		return fmt.Errorf("generate: scanned feature metadata not found in .repox/conventions.json; run `repox scan` before generating")
	}
	return nil
}

func (p *ProjectProfile) Suffixes() map[string]string {
	out := map[string]string{}
	if p == nil || p.conv == nil {
		return out
	}
	for role, rc := range p.conv.Roles {
		if rc.ClassSuffix != "" {
			out[role] = rc.ClassSuffix
		}
	}
	for role, suffix := range legacyClassSuffixes(p.conv.Naming) {
		if _, ok := out[role]; !ok && suffix != "" {
			out[role] = suffix
		}
	}
	return out
}

func (p *ProjectProfile) RoleNames() []string {
	seen := map[string]bool{}
	if p != nil && p.conv != nil {
		for role := range p.conv.Roles {
			seen[role] = true
		}
		for _, role := range p.conv.Naming.SuffixRoles {
			seen[role] = true
		}
		for role := range legacyClassSuffixes(p.conv.Naming) {
			seen[role] = true
		}
	}
	roles := make([]string, 0, len(seen))
	for role := range seen {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func (p *ProjectProfile) RoleSuffix(role string) string {
	if p == nil || p.conv == nil {
		return ""
	}
	if rc, ok := p.conv.Roles[role]; ok && rc.ClassSuffix != "" {
		return rc.ClassSuffix
	}
	return p.Suffixes()[role]
}

func (p *ProjectProfile) RoleForFile(filename string) string {
	ext := filepath.Ext(filename)
	if ext != ".dart" && ext != ".go" {
		return ""
	}
	base := strings.ToLower(strings.TrimSuffix(filename, ext))
	type entry struct {
		suffix string
		role   string
	}
	var entries []entry
	if p != nil && p.conv != nil {
		for role, rc := range p.conv.Roles {
			if rc.FileSuffix != "" {
				entries = append(entries, entry{strings.ToLower(rc.FileSuffix), role})
			}
		}
		for suffix, role := range p.conv.Naming.SuffixRoles {
			entries = append(entries, entry{strings.ToLower(suffix), role})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return len(entries[i].suffix) > len(entries[j].suffix) })
	for _, e := range entries {
		if e.suffix != "" && (base == e.suffix || strings.HasSuffix(base, "_"+e.suffix)) {
			return e.role
		}
	}
	return ""
}

func (p *ProjectProfile) RoleForFeaturePath(featureName, path string) string {
	if role := p.RoleForFile(filepath.Base(path)); role != "" {
		return role
	}
	ext := filepath.Ext(path)
	if ext != ".dart" && ext != ".go" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), ext)
	featureSnake := ToSnakeCase(featureName)
	if strings.HasPrefix(base, featureSnake+"_") {
		return strings.Trim(strings.TrimPrefix(base, featureSnake+"_"), "_")
	}
	return ""
}

func (p *ProjectProfile) OutputPath(tmplPath string, target TargetFeature) string {
	base := filepath.Base(tmplPath)
	name := strings.TrimSuffix(base, ".tmpl")
	role := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	return p.FeatureFilePath(target, role, ext)
}

func (p *ProjectProfile) FeatureFilePath(target TargetFeature, role, ext string) string {
	root := p.conv.FeatureRoot
	if strings.Contains(role, "test") {
		root = p.conv.TestRoot
	}
	return filepath.Join(root, target.Path, p.RouteForRole(target.Path, role), p.FileNameForRole(target, role, ext))
}

func (p *ProjectProfile) FileNameForRole(target TargetFeature, role, ext string) string {
	if ext == "" {
		ext = ".dart"
	}
	suffix := role
	if p != nil && p.conv != nil {
		if rc, ok := p.conv.Roles[role]; ok && rc.FileSuffix != "" {
			suffix = rc.FileSuffix
		}
	}
	return target.Snake + "_" + suffix + ext
}

func (p *ProjectProfile) RouteForRole(featurePath, role string) string {
	pattern := p.conv.FeatureStructure
	if feature, ok := p.FindFeatureConvention(featurePath); ok {
		if feature.FileRoutes != nil {
			if route, exists := feature.FileRoutes[role]; exists {
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
	if p.conv.PatternMappings != nil {
		if mapping, ok := p.conv.PatternMappings[pattern]; ok && mapping.FileRoutes != nil {
			return mapping.FileRoutes[role]
		}
	}
	return ""
}

func (p *ProjectProfile) ImportFor(fromPath string, target TargetFeature, role string) string {
	toPath := p.FeatureFilePath(target, role, ".dart")
	rel, err := filepath.Rel(filepath.Dir(fromPath), toPath)
	if err != nil {
		return filepath.ToSlash(toPath)
	}
	return filepath.ToSlash(rel)
}

func (p *ProjectProfile) FindFeatureConvention(featurePath string) (models.FeatureAnalysis, bool) {
	target := filepath.ToSlash(featurePath)
	targetParent := filepath.ToSlash(filepath.Dir(target))
	if targetParent == "." {
		targetParent = ""
	}
	var sibling models.FeatureAnalysis
	hasSibling := false
	for _, feature := range p.conv.FeaturesAnalysis.Features {
		relPath := FeaturePathWithoutRoot(p.conv.FeatureRoot, feature.Path)
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

func (p *ProjectProfile) FeatureFiles(feature models.FeatureAnalysis, baseDir string) map[string]string {
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
		if isGeneratedSourceFile(path) {
			return nil
		}
		role := p.RoleForFeaturePath(filepath.Base(feature.Path), path)
		if role == "" {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if hasPath(files, rel) {
			return nil
		}
		files[uniqueRole(role, rel, files)] = rel
		return nil
	})
	return files
}

func (p *ProjectProfile) RenamePlan(like models.FeatureAnalysis, target TargetFeature, baseDir string) FeatureRenamePlan {
	files := p.FeatureFiles(like, baseDir)
	sourcePath := FeaturePathWithoutRoot(p.conv.FeatureRoot, like.Path)
	sourceLeaf := filepath.Base(sourcePath)
	sourceSnake := ToSnakeCase(sourceLeaf)
	plan := FeatureRenamePlan{
		SourcePath:        filepath.ToSlash(sourcePath),
		TargetPath:        filepath.ToSlash(target.Path),
		SourceSnake:       sourceSnake,
		TargetSnake:       target.Snake,
		SourcePascal:      ToPascalCase(sourceLeaf),
		TargetPascal:      target.Pascal,
		SourceCamel:       ToCamelCase(sourceLeaf),
		TargetCamel:       target.Camel,
		Files:             files,
		FileTargets:       map[string]string{},
		IdentifierTargets: map[string]string{},
	}
	for role, source := range files {
		ext := filepath.Ext(source)
		targetPath := filepath.ToSlash(p.FeatureFilePath(target, role, ext))
		plan.FileTargets[role] = targetPath
		sourceBase := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		targetBase := strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath))
		sourceIdent := ToPascalCase(sourceBase)
		if suffix := p.RoleSuffix(role); suffix != "" && !strings.HasSuffix(sourceIdent, suffix) {
			sourceIdent += suffix
		}
		targetIdent := ToPascalCase(targetBase)
		if sourceIdent != "" {
			plan.IdentifierTargets[sourceIdent] = targetIdent
			plan.IdentifierTargets[lowerFirst(sourceIdent)] = lowerFirst(targetIdent)
		}
	}
	return plan
}

type FeatureRenamePlan struct {
	SourcePath        string
	TargetPath        string
	SourceSnake       string
	TargetSnake       string
	SourcePascal      string
	TargetPascal      string
	SourceCamel       string
	TargetCamel       string
	Files             map[string]string
	FileTargets       map[string]string
	IdentifierTargets map[string]string
}

func (p FeatureRenamePlan) RewritePath(path string, baseDir string) string {
	if filepath.IsAbs(path) {
		if rel, err := filepath.Rel(baseDir, path); err == nil {
			path = rel
		}
	}
	normalized := filepath.ToSlash(path)
	for role, source := range p.Files {
		if filepath.ToSlash(source) == normalized {
			if target := p.FileTargets[role]; target != "" {
				return target
			}
		}
	}
	return p.Apply(normalized)
}

func (p FeatureRenamePlan) Replacements() map[string]string {
	replacements := map[string]string{
		p.SourcePath:               p.TargetPath,
		displayName(p.SourceSnake): displayName(p.TargetSnake),
	}
	for role, source := range p.Files {
		target := p.FileTargets[role]
		if target == "" {
			continue
		}
		replacements[filepath.ToSlash(source)] = filepath.ToSlash(target)
		sourceBase := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		targetBase := strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
		if sourceBase != "" {
			replacements[sourceBase] = targetBase
		}
	}
	for source, target := range p.IdentifierTargets {
		replacements[source] = target
	}
	return replacements
}

func (p FeatureRenamePlan) Apply(content string) string {
	replacements := p.Replacements()
	keys := make([]string, 0, len(replacements))
	for key, value := range replacements {
		if key != "" && key != value {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, key := range keys {
		content = strings.ReplaceAll(content, key, replacements[key])
	}
	return content
}

func FeaturePathWithoutRoot(featureRoot, path string) string {
	featureRoot = filepath.ToSlash(strings.Trim(featureRoot, "/"))
	path = filepath.ToSlash(strings.Trim(path, "/"))
	if featureRoot != "" && strings.HasPrefix(path, featureRoot+"/") {
		return strings.TrimPrefix(path, featureRoot+"/")
	}
	return path
}

func legacyClassSuffixes(naming models.NamingConvention) map[string]string {
	out := map[string]string{}
	if len(naming.SuffixRoles) > 0 {
		for suffix, role := range naming.SuffixRoles {
			out[role] = ToPascalCase(suffix)
		}
	}
	add := func(role, suffix string) {
		if suffix != "" {
			out[role] = suffix
		}
	}
	add("screen", naming.ScreenSuffix)
	add("bloc", naming.BlocSuffix)
	add("event", naming.EventSuffix)
	add("state", naming.StateSuffix)
	add("repository", naming.RepositorySuffix)
	add("usecase", naming.UsecaseSuffix)
	add("handler", naming.HandlerSuffix)
	add("service", naming.ServiceSuffix)
	return out
}

func isGeneratedSourceFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".g.dart") ||
		strings.HasSuffix(base, ".freezed.dart") ||
		strings.HasSuffix(base, ".gen.dart") ||
		strings.HasSuffix(base, ".generated.dart")
}

func hasPath(files map[string]string, path string) bool {
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

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToLower(s[:1]) + s[1:]
}
