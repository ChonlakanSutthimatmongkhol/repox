package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/ai"
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

	result := fmt.Sprintf("Scanned repository:\n  Project type: %s\n  Feature root: %s\n  Structure: %s\n  State management: %s\n  Routing: %s\n  Indexed features: %d\n\nConventions saved to .repox/conventions.json",
		conv.ProjectType, conv.FeatureRoot, conv.FeatureStructure, conv.StateManagement, conv.Routing.Type, len(examples))
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
	useAI := optionalBool(args, "use_ai")
	useExamples := optionalBool(args, "use_examples")
	force := optionalBool(args, "force")
	dryRun := optionalBool(args, "dry_run")

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

	tmplName := cfg.DefaultTemplate
	var files []generator.GeneratedFile
	genMode := "template"

	if useAI {
		genMode = "ai"
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return callError(fmt.Errorf("set ANTHROPIC_API_KEY environment variable"))
		}

		tplGen := generator.NewTemplateGenerator()
		tplFiles, err := tplGen.Generate(featureName, tmplName, &conv)
		if err != nil {
			return callError(err)
		}
		targetPaths := make([]string, len(tplFiles))
		for i, f := range tplFiles {
			targetPaths[i] = f.Path
		}

		var examples []models.Example
		if useExamples {
			examples, _ = config.Load[[]models.Example](config.RepoxPath("examples.json"))
			if len(examples) == 0 {
				examples, _ = retriever.IndexFeatures(cwd, &conv)
			}
		}
		similar := retriever.FindSimilar(featureName, examples, 3)
		lessons, _ := config.Load[[]models.Lesson](config.RepoxPath("lessons.json"))

		client := ai.NewAnthropicClient(apiKey, cfg.AI.GenerationModel)
		aiResp, err := client.Generate(ai.GenerateRequest{
			FeatureName:    featureName,
			Conventions:    &conv,
			Examples:       similar,
			Lessons:        lessons,
			TargetFiles:    targetPaths,
			TargetTemplate: tmplName,
			RootDir:        cwd,
		})
		if err != nil {
			return callError(err)
		}
		files = make([]generator.GeneratedFile, len(aiResp.Files))
		for i, f := range aiResp.Files {
			files[i] = generator.GeneratedFile{Path: f.Path, Content: f.Content}
		}
	} else {
		gen := generator.NewTemplateGenerator()
		files, err = gen.Generate(featureName, tmplName, &conv)
		if err != nil {
			return callError(err)
		}
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
	snapDir := ""
	if genMode == "ai" {
		snapDir = saveSnapshotMCP(genID, files, cwd)
	}
	existing, _ := config.Load[[]models.Generation](config.RepoxPath("generations.json"))
	existing = append(existing, models.Generation{
		ID: genID, FeatureName: featureName, Template: tmplName,
		Mode: genMode, Files: filePaths, SnapshotDir: snapDir, CreatedAt: time.Now(),
	})
	_ = config.Save(config.RepoxPath("generations.json"), existing)

	_ = writtenAbs // available for formatter if needed
	return callResult(fmt.Sprintf("%d created, %d skipped\n\n%s", written, skipped, strings.Join(lines, "\n")))
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
	return callResult("repox learn is available via CLI: `repox learn [--from <gen_id>] [--approve] [--list]`\n\nLesson extraction requires API access and interactive review — run it from the terminal.")
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
