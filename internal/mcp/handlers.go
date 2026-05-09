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
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/retriever"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/scanner"
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
		fmt.Fprintln(&b, "\nNext step:\n  repox generate feature <name>")
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
	fmt.Fprintln(&b, "\nNext step:\n  repox generate feature <name>")
	return b.String()
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
