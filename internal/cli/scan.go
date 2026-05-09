package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	fmt.Fprintf(out, "  State management:  %s\n", conv.StateManagement)
	fmt.Fprintf(out, "  Feature root:      %s\n", conv.FeatureRoot)
	fmt.Fprintf(out, "  Feature structure: %s\n", conv.FeatureStructure)
	fmt.Fprintf(out, "  Test root:         %s\n", conv.TestRoot)
	fmt.Fprintln(out, "  Naming:")
	fmt.Fprintf(out, "    Screen suffix:   %s\n", conv.Naming.ScreenSuffix)
	fmt.Fprintf(out, "    Bloc suffix:     %s\n", conv.Naming.BlocSuffix)
	fmt.Fprintf(out, "    Event suffix:    %s\n", conv.Naming.EventSuffix)
	fmt.Fprintf(out, "    State suffix:    %s\n", conv.Naming.StateSuffix)
	fmt.Fprintf(out, "  Routing:           %s (%s)\n", conv.Routing.Type, conv.Routing.RouteFile)
	fmt.Fprintf(out, "  Common imports:    %d detected\n", len(conv.CommonImports))

	// Analyze patterns in existing features
	cwd, err := os.Getwd()
	if err == nil && conv.FeatureRoot != "" {
		featureRootPath := filepath.Join(cwd, conv.FeatureRoot)
		patterns := analyzeFeaturePatterns(featureRootPath)
		if len(patterns) > 0 {
			printPatternAnalysis(out, patterns)
		}
	}
}

// analyzeFeaturePatterns scans all features and returns pattern distribution
func analyzeFeaturePatterns(featureRootPath string) map[string]int {
	patterns := make(map[string]int)

	entries, err := os.ReadDir(featureRootPath)
	if err != nil {
		return patterns
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		featurePath := filepath.Join(featureRootPath, entry.Name())
		pattern := detectFeatureStructure(featurePath)
		patterns[pattern]++
	}

	return patterns
}

// detectFeatureStructure checks a feature directory structure
func detectFeatureStructure(featurePath string) string {
	entries, err := os.ReadDir(featurePath)
	if err != nil {
		return "flat"
	}

	hasPresentation := false
	hasDomain := false
	hasData := false
	hasBloc := false
	hasScreen := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		switch entry.Name() {
		case "presentation":
			hasPresentation = true
		case "domain":
			hasDomain = true
		case "data":
			hasData = true
		case "bloc":
			hasBloc = true
		case "screen":
			hasScreen = true
		}
	}

	if hasPresentation && (hasDomain || hasData) {
		return "clean_architecture"
	}
	if hasBloc || hasScreen {
		return "grouped"
	}
	return "flat"
}

// printPatternAnalysis displays pattern distribution and recommendations
func printPatternAnalysis(out io.Writer, patterns map[string]int) {
	if len(patterns) == 0 {
		return
	}

	total := 0
	for _, count := range patterns {
		total += count
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Pattern Analysis:")

	// Find recommended pattern (most frequent)
	var recommended string
	maxCount := 0
	for pattern, count := range patterns {
		percentage := (count * 100) / total
		if percentage > 0 {
			fmt.Fprintf(out, "  %s: %d features (%d%%)\n", pattern, count, percentage)
		}
		if count > maxCount {
			maxCount = count
			recommended = pattern
		}
	}

	fmt.Fprintf(out, "\nRecommended pattern: %s\n", recommended)
	fmt.Fprintln(out, "\nTo generate a new feature, run:")
	fmt.Fprintln(out, "  repox generate feature <feature_name>")
}
