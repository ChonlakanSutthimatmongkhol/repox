package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/mapgen"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/output"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/retriever"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/scanner"
	projectskill "github.com/ChonlakanSutthimatmongkhol/repox/internal/skill"
)

func handleScan(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := argsMap(req.Params.Arguments)
	cwd, err := os.Getwd()
	if err != nil {
		return callError(err)
	}
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox init first"))
	}

	projectOverride := optionalString(args, "project_override")
	projectType := projectOverride
	if projectType == "" {
		projectType, err = scanner.DetectProjectType(cwd)
		if err != nil {
			return callError(err)
		}
	}

	var s scanner.Scanner
	switch projectType {
	case "flutter", "dart":
		s = &scanner.FlutterScanner{}
	default:
		return callError(fmt.Errorf("unsupported project type %q", projectType))
	}

	conv, err := s.Scan(cwd)
	if err != nil {
		return callError(err)
	}
	if err := config.Save(config.RepoxPath("conventions.json"), conv); err != nil {
		return callError(err)
	}

	examples, _ := retriever.IndexFeatures(cwd, conv)
	if len(examples) > 0 {
		_ = config.Save(config.RepoxPath("examples.json"), examples)
	}

	result := fmt.Sprintf("Scanned repository:\n  Project type: %s\n  Feature root: %s\n  Structure: %s\n  State management: %s\n  Routing: %s\n  Indexed features: %d\n%s\n\nConventions saved to .repox/conventions.json",
		conv.ProjectType, conv.FeatureRoot, conv.FeatureStructure, conv.StateManagement, conv.Routing.Type, len(examples), formatPatternAnalysis(conv.FeaturesAnalysis))
	return callResult(result)
}

func handleGenerate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := argsMap(req.Params.Arguments)
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox init first"))
	}

	featureName := optionalString(args, "feature_name")
	if featureName == "" {
		return callError(fmt.Errorf("feature_name is required"))
	}
	force := optionalBool(args, "force")
	dryRun := optionalBool(args, "dry_run")
	patternOverride := optionalString(args, "pattern")

	cwd, err := os.Getwd()
	if err != nil {
		return callError(err)
	}
	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	if err != nil {
		return callError(err)
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return callError(err)
	}
	if err := applyPatternOverride(&conv, patternOverride); err != nil {
		return callError(err)
	}

	tmplName := cfg.DefaultTemplate
	var files []generator.GeneratedFile
	genMode := "template"

	gen := generator.NewTemplateGenerator()
	files, err = gen.Generate(featureName, tmplName, &conv)
	if err != nil {
		return callError(err)
	}

	if dryRun {
		var paths []string
		for _, f := range files {
			paths = append(paths, "  "+f.Path)
		}
		return callResult("Dry run — would generate:\n" + strings.Join(paths, "\n"))
	}

	results, err := generator.WriteFiles(files, cwd, force)
	if err != nil {
		return callError(err)
	}

	written, skipped := 0, 0
	var lines, filePaths []string
	var writtenAbs []string
	for _, r := range results {
		filePaths = append(filePaths, r.Path)
		if r.Written {
			lines = append(lines, "  created "+r.Path)
			writtenAbs = append(writtenAbs, filepath.Join(cwd, r.Path))
			written++
		} else {
			lines = append(lines, "  skipped "+r.Reason)
			skipped++
		}
	}

	genID := fmt.Sprintf("gen_%d", time.Now().Unix())
	snapDir := saveSnapshotMCP(genID, files, cwd)
	existing, _ := config.Load[[]models.Generation](config.RepoxPath("generations.json"))
	existing = append(existing, models.Generation{
		ID: genID, FeatureName: featureName, Template: tmplName,
		Mode: genMode, Files: filePaths, SnapshotDir: snapDir, CreatedAt: time.Now(),
	})
	_ = config.Save(config.RepoxPath("generations.json"), existing)

	_ = writtenAbs // available for formatter if needed
	return callResult(fmt.Sprintf("%d created, %d skipped\n\n%s", written, skipped, strings.Join(lines, "\n")))
}

func applyPatternOverride(conv *models.Convention, override string) error {
	pattern := conv.FeatureStructure
	if conv.FeaturesAnalysis.RecommendedPattern != "" {
		pattern = conv.FeaturesAnalysis.RecommendedPattern
	}
	if override != "" {
		pattern = override
	}
	if pattern == "" {
		pattern = "flat"
	}
	switch pattern {
	case "flat", "grouped", "clean_architecture":
		conv.FeatureStructure = pattern
		return nil
	default:
		return fmt.Errorf("unsupported pattern %q (use flat, grouped, or clean_architecture)", pattern)
	}
}

func formatPatternAnalysis(analysis models.FeaturesAnalysis) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n\nDiscover:\n  Features found: %d\n\nPattern Analysis:\n  Total features: %d\n", len(analysis.Features), len(analysis.Features))
	if len(analysis.Features) == 0 {
		fmt.Fprintln(&b, "  No feature folders found to analyze.")
		fmt.Fprintln(&b, "\nNext step:\n  repox generate feature <name-or-path>")
		return b.String()
	}
	fmt.Fprintln(&b, "  Pattern distribution:")
	patterns := make([]string, 0, len(analysis.PatternDistribution))
	for pattern := range analysis.PatternDistribution {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	for _, pattern := range patterns {
		item := analysis.PatternDistribution[pattern]
		fmt.Fprintf(&b, "    %s: %d features (%.1f%%)\n", pattern, item.Count, item.Percentage)
	}
	fmt.Fprintf(&b, "  Recommended pattern: %s\n", analysis.RecommendedPattern)
	fmt.Fprintf(&b, "  Latest pattern:      %s\n", analysis.LatestPattern)
	formatNestedFlows(&b, analysis.Features)
	formatRoleAnatomy(&b, analysis.RoleAnatomy)
	fmt.Fprintln(&b, "\nNext step:\n  repox generate feature <name-or-path>")
	return b.String()
}

func formatNestedFlows(b *strings.Builder, features []models.FeatureAnalysis) {
	var nested []models.FeatureAnalysis
	for _, feature := range features {
		if feature.Parent != "" || feature.Depth > 1 {
			nested = append(nested, feature)
		}
	}
	if len(nested) == 0 {
		return
	}
	fmt.Fprintf(b, "  Nested flows:        %d\n", len(nested))
	limit := len(nested)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		feature := nested[i]
		roles := make([]string, 0, len(feature.Files))
		for role := range feature.Files {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		fmt.Fprintf(b, "    - %s [%s]\n", feature.Path, strings.Join(roles, ", "))
	}
}

func formatRoleAnatomy(b *strings.Builder, roleAnatomy map[string]models.RoleAnatomy) {
	if len(roleAnatomy) == 0 {
		return
	}
	roles := make([]string, 0, len(roleAnatomy))
	for role := range roleAnatomy {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	fmt.Fprintf(b, "  Role anatomy:       %d roles\n", len(roles))
	limit := len(roles)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		role := roles[i]
		anatomy := roleAnatomy[role]
		fmt.Fprintf(b, "    - %s: %d files", role, anatomy.FeatureCount)
		if len(anatomy.BaseClasses) > 0 {
			fmt.Fprintf(b, ", base %s", anatomy.BaseClasses[0].Name)
		}
		if len(anatomy.Methods) > 0 {
			fmt.Fprintf(b, ", methods %s", strings.Join(anatomyVoteNames(anatomy.Methods, 3), ", "))
		}
		fmt.Fprintln(b)
	}
}

func anatomyVoteNames(votes []models.AnatomyVote, limit int) []string {
	if len(votes) < limit {
		limit = len(votes)
	}
	names := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		names = append(names, votes[i].Name)
	}
	return names
}

func saveSnapshotMCP(genID string, files []generator.GeneratedFile, baseDir string) string {
	snapDir := config.RepoxPath("snapshots/" + genID)
	for _, f := range files {
		dest := filepath.Join(snapDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}
		_ = os.WriteFile(dest, []byte(f.Content), 0o644)
	}
	abs, err := filepath.Abs(snapDir)
	if err != nil {
		return snapDir
	}
	return abs
}

func handleFindSimilar(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := argsMap(req.Params.Arguments)
	featureName := optionalString(args, "feature_name")
	if featureName == "" {
		return callError(fmt.Errorf("feature_name is required"))
	}
	topN := optionalInt(args, "top_n", "3")
	if topN <= 0 {
		topN = 3
	}
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox init first"))
	}

	cwd, err := os.Getwd()
	if err != nil {
		return callError(err)
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return callError(err)
	}

	examples, _ := config.Load[[]models.Example](config.RepoxPath("examples.json"))
	if len(examples) == 0 {
		examples, _ = retriever.IndexFeatures(cwd, &conv)
	}

	similar := retriever.FindSimilar(featureName, examples, topN)
	if len(similar) == 0 {
		return callResult("No similar features found.")
	}

	var lines []string
	for _, ex := range similar {
		var comps []string
		if ex.Metadata.HasBloc {
			comps = append(comps, "bloc")
		}
		if ex.Metadata.HasScreen {
			comps = append(comps, "screen")
		}
		if ex.Metadata.HasRepository {
			comps = append(comps, "repository")
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s) [%s]", ex.Name, ex.Path, strings.Join(comps, ", ")))
	}
	return callResult("Similar features:\n" + strings.Join(lines, "\n"))
}

func handleLearn(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return callResult("repox learn is available via CLI: `repox learn [--from <gen_id>] [--approve] [--list]`\n\nOffline learning reads local generation snapshots and developer edits, then stores approved lessons in .repox/lessons.json. Run `repox skill generate` afterward to refresh the project skill.")
}

func handleExplainConvention(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox init first"))
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return callError(fmt.Errorf("run `repox scan` first to detect conventions"))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Repository Conventions:\n\n")
	fmt.Fprintf(&b, "Project type:      %s\n", conv.ProjectType)
	fmt.Fprintf(&b, "State management:  %s\n", conv.StateManagement)
	fmt.Fprintf(&b, "Feature root:      %s\n", conv.FeatureRoot)
	fmt.Fprintf(&b, "Feature structure: %s\n", conv.FeatureStructure)
	fmt.Fprintf(&b, "Test root:         %s\n", conv.TestRoot)
	fmt.Fprintf(&b, "\nNaming:\n")
	fmt.Fprintf(&b, "  Screen suffix:     %s\n", conv.Naming.ScreenSuffix)
	fmt.Fprintf(&b, "  Bloc suffix:       %s\n", conv.Naming.BlocSuffix)
	fmt.Fprintf(&b, "  Repository suffix: %s\n", conv.Naming.RepositorySuffix)
	fmt.Fprintf(&b, "  UseCase suffix:    %s\n", conv.Naming.UsecaseSuffix)
	fmt.Fprintf(&b, "\nRouting: %s (%s)\n", conv.Routing.Type, conv.Routing.RouteFile)
	fmt.Fprintf(&b, "Top imports: %s\n", strings.Join(conv.CommonImports, ", "))
	return callResult(b.String())
}

func handleSetup(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := ensureRepoxInit(); err != nil {
		return callError(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return callError(err)
	}
	projectType, err := scanner.DetectProjectType(cwd)
	if err != nil {
		return callError(err)
	}
	var s scanner.Scanner
	switch projectType {
	case "flutter", "dart":
		s = &scanner.FlutterScanner{}
	case "go":
		s = &scanner.GoScanner{}
	default:
		return callError(fmt.Errorf("unsupported project type %q", projectType))
	}
	conv, err := s.Scan(cwd)
	if err != nil {
		return callError(err)
	}
	if err := config.Save(config.RepoxPath("conventions.json"), conv); err != nil {
		return callError(err)
	}
	examples, _ := retriever.IndexFeatures(cwd, conv)
	_ = config.Save(config.RepoxPath("examples.json"), examples)
	cfg, _ := config.Load[models.Config](config.RepoxPath("config.json"))
	cfg.ProjectType = conv.ProjectType
	cfg.FeatureRoot = conv.FeatureRoot
	cfg.TestRoot = conv.TestRoot
	cfg.DefaultTemplate = defaultMCPTemplate(conv.ProjectType)
	_ = config.Save(config.RepoxPath("config.json"), cfg)
	skillInput := projectskill.Input{Config: cfg, Convention: *conv, Examples: examples}
	if err := mcpWriteTextFile(config.RepoxPath(filepath.Join("skill", "SKILL.md")), projectskill.BuildProjectSkill(skillInput)); err != nil {
		return callError(err)
	}
	return callResult(output.Contract(
		"Repox setup complete.",
		output.BulletList([]string{"Project type: " + conv.ProjectType, "Feature root: " + conv.FeatureRoot}),
		output.BulletList([]string{fmt.Sprintf("Features indexed: %d", len(examples)), "Generated .repox/skill/SKILL.md"}),
		output.BulletList(firstMCPFeatures(conv, 5)),
		[]string{"repox_doctor", "repox_map", "repox_explain"},
		nil,
	))
}

func handleDoctor(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var warnings []string
	var findings []string
	if !config.RepoxDirExists() {
		warnings = append(warnings, ".repox missing. Run repox_setup or repox setup.")
		return callResult(output.Contract("Repox doctor diagnosis.", "", output.BulletList(findings), "", []string{"repox_setup"}, warnings))
	}
	findings = append(findings, ".repox exists")
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		warnings = append(warnings, "conventions.json missing or unreadable. Run repox_scan.")
	} else {
		findings = append(findings, "Project type: "+conv.ProjectType)
		findings = append(findings, fmt.Sprintf("Features indexed: %d", len(conv.FeaturesAnalysis.Features)))
	}
	if _, err := os.Stat(config.RepoxPath(filepath.Join("skill", "SKILL.md"))); err != nil {
		warnings = append(warnings, "Skill file missing. Run repox_setup or repox skill generate.")
	}
	if _, err := os.Stat(config.RepoxPath(filepath.Join("maps", "project.md"))); err != nil {
		warnings = append(warnings, "Map not generated. Run repox_map or repox map.")
	}
	return callResult(output.Contract("Repox doctor diagnosis.", "", output.BulletList(findings), "", []string{"repox_setup", "repox_map", "repox_explain"}, warnings))
}

func handleMap(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := argsMap(req.Params.Arguments)
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox_setup first"))
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return callError(fmt.Errorf("run repox_scan first to detect conventions"))
	}
	files, err := writeMCPMaps(conv, optionalString(args, "feature"))
	if err != nil {
		return callError(err)
	}
	return callResult(output.Contract(
		"Generated Repox project maps.",
		output.BulletList([]string{"Project type: " + conv.ProjectType, "Feature root: " + conv.FeatureRoot}),
		output.BulletList([]string{fmt.Sprintf("Features indexed: %d", len(conv.FeaturesAnalysis.Features)), "Generated files:\n" + output.BulletList(files)}),
		output.BulletList(firstMCPFeatures(&conv, 5)),
		[]string{"repox_explain", "repox_generate"},
		[]string{"Run repox_scan if maps look outdated."},
	))
}

func handleExplain(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := argsMap(req.Params.Arguments)
	if !config.RepoxDirExists() {
		return callError(fmt.Errorf("no .repox/ directory found — run repox_setup first"))
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return callError(fmt.Errorf("run repox_scan first to detect conventions"))
	}
	featureName := optionalString(args, "feature")
	role := optionalString(args, "role")
	var warnings []string
	var findings []string
	findings = append(findings, fmt.Sprintf("Features indexed: %d", len(conv.FeaturesAnalysis.Features)))
	if featureName != "" {
		if feature, ok := mapgen.FindFeature(conv, featureName); ok {
			findings = append(findings, "Selected feature: "+feature.Path)
		} else {
			warnings = append(warnings, "Feature "+featureName+" not found in scanned conventions.")
		}
	}
	if role != "" {
		if _, ok := conv.FeaturesAnalysis.RoleAnatomy[role]; ok {
			findings = append(findings, "Selected role: "+role)
		} else {
			warnings = append(warnings, "Role "+role+" not found in scanned anatomy.")
		}
	}
	return callResult(output.Contract(
		"Repox convention explanation.",
		output.BulletList([]string{
			"Project type: " + conv.ProjectType,
			"Feature root: " + conv.FeatureRoot,
			"State management: " + conv.StateManagement,
			"Routing: " + conv.Routing.Type,
		}),
		output.BulletList(findings),
		output.BulletList(firstMCPFeatures(&conv, 5)),
		[]string{"repox_map", "repox_generate"},
		append(warnings, "Run repox_scan if conventions look outdated."),
	))
}

func ensureRepoxInit() error {
	if err := os.MkdirAll(".repox", 0o755); err != nil {
		return err
	}
	files := map[string]any{
		"config.json":      config.DefaultConfig(),
		"conventions.json": config.DefaultConventions(),
		"examples.json":    []models.Example{},
		"lessons.json":     []models.Lesson{},
		"generations.json": []models.Generation{},
	}
	for name, value := range files {
		path := config.RepoxPath(name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := config.Save(path, value); err != nil {
			return err
		}
	}
	return nil
}

func mcpWriteTextFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeMCPMaps(conv models.Convention, feature string) ([]string, error) {
	maps := []mapgen.GeneratedMap{
		{Path: config.RepoxPath(filepath.Join("maps", "project.md")), Content: mapgen.GenerateProjectMarkdown(conv)},
		{Path: config.RepoxPath(filepath.Join("maps", "conventions.md")), Content: mapgen.GenerateConventionsMarkdown(conv)},
		{Path: config.RepoxPath(filepath.Join("maps", "project-markmap.md")), Content: mapgen.GenerateProjectMarkmap(conv)},
	}
	if feature != "" {
		if name, content, ok := mapgen.GenerateFeatureMarkdown(conv, feature); ok {
			maps = append(maps, mapgen.GeneratedMap{Path: config.RepoxPath(filepath.Join("maps", "features", name+".md")), Content: content})
		} else {
			return nil, fmt.Errorf("feature %q not found in scanned conventions", feature)
		}
		if name, content, ok := mapgen.GenerateFeatureMarkmap(conv, feature); ok {
			maps = append(maps, mapgen.GeneratedMap{Path: config.RepoxPath(filepath.Join("maps", "features", name+"-markmap.md")), Content: content})
		}
	}
	var files []string
	for _, item := range maps {
		if err := mcpWriteTextFile(item.Path, item.Content); err != nil {
			return nil, err
		}
		files = append(files, item.Path)
	}
	return files, nil
}

func firstMCPFeatures(conv *models.Convention, limit int) []string {
	features := append([]models.FeatureAnalysis(nil), conv.FeaturesAnalysis.Features...)
	sort.Slice(features, func(i, j int) bool { return features[i].Path < features[j].Path })
	if len(features) < limit {
		limit = len(features)
	}
	items := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, features[i].Path)
	}
	return items
}

func defaultMCPTemplate(projectType string) string {
	switch projectType {
	case "go":
		return "go_clean_feature"
	default:
		return "flutter_bloc_feature"
	}
}
