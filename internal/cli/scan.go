package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/retriever"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/scanner"
)

var (
	scanProjectOverride string
	scanDeep            bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan repo and detect conventions",
	Long:  "Analyzes the current repository and writes detected conventions to .repox/conventions.json.",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanProjectOverride, "project", "", "Override project type detection (flutter, go, node)")
	scanCmd.Flags().BoolVar(&scanDeep, "deep", true, "Scan file contents for imports")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("scan: getwd: %w", err)
	}

	projectType := scanProjectOverride
	if projectType == "" {
		projectType, err = scanner.DetectProjectType(cwd)
		if err != nil {
			return fmt.Errorf("scan: detect project type: %w", err)
		}
	}

	var s scanner.Scanner
	switch projectType {
	case "flutter", "dart":
		s = &scanner.FlutterScanner{}
	case "go":
		s = &scanner.GoScanner{}
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: unsupported project type %q — saving partial results\n", projectType)
		conv := &models.Convention{ProjectType: projectType}
		return saveScanResult(cmd, conv)
	}

	conv, err := s.Scan(cwd)
	if err != nil {
		return fmt.Errorf("scan: %w", err)
	}

	return saveScanResult(cmd, conv)
}

func saveScanResult(cmd *cobra.Command, conv *models.Convention) error {
	if err := config.Save(config.RepoxPath("conventions.json"), conv); err != nil {
		return fmt.Errorf("scan: save conventions: %w", err)
	}

	// Update config.json with detected project type and feature root.
	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	if err == nil {
		cfg.ProjectType = conv.ProjectType
		if conv.FeatureRoot != "" {
			cfg.FeatureRoot = conv.FeatureRoot
		}
		if conv.TestRoot != "" {
			cfg.TestRoot = conv.TestRoot
		}
		cfg.DefaultTemplate = defaultTemplateFor(conv.ProjectType)
		_ = config.Save(config.RepoxPath("config.json"), cfg)
	}

	// Index existing features and save examples.json.
	cwd, err := os.Getwd()
	if err == nil {
		examples, idxErr := retriever.IndexFeatures(cwd, conv)
		if idxErr == nil && len(examples) > 0 {
			_ = config.Save(config.RepoxPath("examples.json"), examples)
			fmt.Fprintf(cmd.OutOrStdout(), "  Indexed %d features\n", len(examples))
		}
	}

	printScanSummary(cmd, conv)
	fmt.Fprintln(cmd.OutOrStdout(), "\nConventions saved to .repox/conventions.json")
	return nil
}

func printScanSummary(cmd *cobra.Command, conv *models.Convention) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Scanned repository:")
	fmt.Fprintf(out, "  Project type:      %s\n", conv.ProjectType)
	fmt.Fprintf(out, "  Feature root:      %s\n", conv.FeatureRoot)
	fmt.Fprintf(out, "  Feature structure: %s\n", conv.FeatureStructure)
	fmt.Fprintf(out, "  Test root:         %s\n", conv.TestRoot)

	switch conv.ProjectType {
	case "flutter", "dart":
		fmt.Fprintf(out, "  State management:  %s\n", conv.StateManagement)
		fmt.Fprintln(out, "  Naming:")
		fmt.Fprintf(out, "    Screen suffix:   %s\n", conv.Naming.ScreenSuffix)
		fmt.Fprintf(out, "    Bloc suffix:     %s\n", conv.Naming.BlocSuffix)
		fmt.Fprintf(out, "    Event suffix:    %s\n", conv.Naming.EventSuffix)
		fmt.Fprintf(out, "    State suffix:    %s\n", conv.Naming.StateSuffix)
		fmt.Fprintf(out, "  Routing:           %s (%s)\n", conv.Routing.Type, conv.Routing.RouteFile)
	case "go":
		if conv.ModulePath != "" {
			fmt.Fprintf(out, "  Module:            %s\n", conv.ModulePath)
		}
		fmt.Fprintln(out, "  Naming:")
		fmt.Fprintf(out, "    Handler suffix:  %s\n", conv.Naming.HandlerSuffix)
		fmt.Fprintf(out, "    Service suffix:  %s\n", conv.Naming.ServiceSuffix)
		fmt.Fprintf(out, "    Repo suffix:     %s\n", conv.Naming.RepositorySuffix)
		fmt.Fprintf(out, "  HTTP framework:    %s\n", conv.Routing.Type)
	}

	fmt.Fprintf(out, "  Common imports:    %d detected\n", len(conv.CommonImports))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Discover:")
	fmt.Fprintf(out, "  Feature root:       %s\n", conv.FeatureRoot)
	fmt.Fprintf(out, "  Features found:     %d\n", len(conv.FeaturesAnalysis.Features))
	printPatternAnalysis(out, conv.FeaturesAnalysis)
}

func defaultTemplateFor(projectType string) string {
	switch projectType {
	case "go":
		return "go_clean_feature"
	default:
		return "flutter_bloc_feature"
	}
}

// printPatternAnalysis displays pattern distribution and recommendations
func printPatternAnalysis(out io.Writer, analysis models.FeaturesAnalysis) {
	if len(analysis.Features) == 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Pattern Analysis:")
		fmt.Fprintln(out, "  Total features: 0")
		fmt.Fprintln(out, "  No feature folders found to analyze.")
		fmt.Fprintln(out, "\nNext step:")
		fmt.Fprintln(out, "  repox generate feature <name-or-path>")
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Pattern Analysis:")
	fmt.Fprintf(out, "  Total features: %d\n", len(analysis.Features))

	patterns := make([]string, 0, len(analysis.PatternDistribution))
	for pattern := range analysis.PatternDistribution {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	fmt.Fprintln(out, "  Pattern distribution:")
	for _, pattern := range patterns {
		item := analysis.PatternDistribution[pattern]
		fmt.Fprintf(out, "    %s: %d features (%.1f%%)\n", pattern, item.Count, item.Percentage)
	}

	fmt.Fprintf(out, "  Recommended pattern: %s\n", analysis.RecommendedPattern)
	fmt.Fprintf(out, "  Latest pattern:      %s\n", analysis.LatestPattern)
	printNestedFlowSummary(out, analysis.Features)
	fmt.Fprintln(out, "\nNext step:")
	fmt.Fprintln(out, "  repox generate feature <name-or-path>")
}

func printNestedFlowSummary(out io.Writer, features []models.FeatureAnalysis) {
	var nested []models.FeatureAnalysis
	for _, feature := range features {
		if feature.Parent != "" || feature.Depth > 1 {
			nested = append(nested, feature)
		}
	}
	if len(nested) == 0 {
		return
	}

	fmt.Fprintf(out, "  Nested flows:        %d\n", len(nested))
	limit := len(nested)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		feature := nested[i]
		fmt.Fprintf(out, "    - %s [%s]\n", strings.TrimPrefix(feature.Path, "/"), strings.Join(featureRoles(feature), ", "))
	}
}

func featureRoles(feature models.FeatureAnalysis) []string {
	roles := make([]string, 0, len(feature.Files))
	for role := range feature.Files {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}
