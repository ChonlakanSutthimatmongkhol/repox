package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
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
}
