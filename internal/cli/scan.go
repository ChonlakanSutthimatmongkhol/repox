package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	scanFeatureRoot     string
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
	scanCmd.Flags().StringVar(&scanFeatureRoot, "feature-root", "", "Override feature root path (e.g. internal/customer)")
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

	if strings.TrimSpace(scanFeatureRoot) != "" {
		if err := applyFeatureRootOverride(conv, cwd, scanFeatureRoot); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: --feature-root override failed: %v\n", err)
		}
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
	printRoleAnatomySummary(out, analysis.RoleAnatomy)
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

func printRoleAnatomySummary(out io.Writer, roleAnatomy map[string]models.RoleAnatomy) {
	if len(roleAnatomy) == 0 {
		return
	}
	roles := make([]string, 0, len(roleAnatomy))
	for role := range roleAnatomy {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	fmt.Fprintf(out, "  Role anatomy:       %d roles\n", len(roles))
	limit := len(roles)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		role := roles[i]
		anatomy := roleAnatomy[role]
		fmt.Fprintf(out, "    - %s: %d files", role, anatomy.FeatureCount)
		if len(anatomy.BaseClasses) > 0 {
			fmt.Fprintf(out, ", base %s", anatomy.BaseClasses[0].Name)
		}
		if len(anatomy.Methods) > 0 {
			fmt.Fprintf(out, ", methods %s", strings.Join(anatomyVoteNames(anatomy.Methods, 3), ", "))
		}
		fmt.Fprintln(out)
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

// applyFeatureRootOverride restricts the scanned features to only those under
// featureRoot. If featureRoot points to a single feature folder (e.g. internal/customer),
// it uses the parent as the scan root and filters to matching features only.
func applyFeatureRootOverride(conv *models.Convention, rootDir, featureRoot string) error {
	// Try featureRoot directly as a container of features.
	analysis, err := scanner.AnalyzeFeatureRoot(rootDir, featureRoot, conv.Naming)
	if err != nil || len(analysis.Features) == 0 {
		// Fallback: treat featureRoot as a single feature — scan its parent and filter.
		parent := filepath.Dir(featureRoot)
		if parent == "." {
			parent = featureRoot
		}
		analysis, err = scanner.AnalyzeFeatureRoot(rootDir, parent, conv.Naming)
		if err != nil {
			return err
		}
		// Keep only features whose path starts with featureRoot.
		var filtered []models.FeatureAnalysis
		for _, f := range analysis.Features {
			if f.Path == featureRoot || strings.HasPrefix(f.Path, featureRoot+"/") {
				filtered = append(filtered, f)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no features found under %q", featureRoot)
		}
		analysis.Features = filtered
	}

	conv.FeatureRoot = featureRoot
	conv.FeaturesAnalysis = analysis
	if analysis.RecommendedPattern != "" {
		conv.FeatureStructure = analysis.RecommendedPattern
	}
	conv.PatternMappings = scanner.InferPatternMappings(analysis.Features, conv.PatternMappings)
	return nil
}
