package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/mapgen"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/output"
)

var (
	explainFeature string
	explainRole    string
	explainAI      bool
)

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Explain detected Repox conventions",
	RunE:  runExplain,
}

func init() {
	explainCmd.Flags().StringVar(&explainFeature, "feature", "", "Explain a scanned feature")
	explainCmd.Flags().StringVar(&explainRole, "role", "", "Explain a scanned role")
	explainCmd.Flags().BoolVar(&explainAI, "ai", false, "Print compact AI-friendly markdown")
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox setup` first")
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("explain: load conventions: run `repox scan` first: %w", err)
	}
	if explainAI {
		fmt.Fprint(cmd.OutOrStdout(), buildExplainAI(conv, explainFeature, explainRole))
		return nil
	}
	fmt.Fprint(cmd.OutOrStdout(), buildExplainHuman(conv, explainFeature, explainRole))
	return nil
}

func buildExplainHuman(conv models.Convention, featureName, role string) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Repox Convention Explanation")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Project:")
	fmt.Fprintf(&b, "- Type: %s\n", valueOrUnknown(conv.ProjectType))
	fmt.Fprintf(&b, "- Feature root: %s\n", valueOrUnknown(conv.FeatureRoot))
	fmt.Fprintf(&b, "- State management: %s\n", valueOrUnknown(conv.StateManagement))
	fmt.Fprintf(&b, "- Routing: %s\n", valueOrUnknown(conv.Routing.Type))
	fmt.Fprintf(&b, "- Recommended pattern: %s\n", recommendedPatternForExplain(conv))
	if role != "" {
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "Role: %s\n", role)
		if anatomy, ok := conv.FeaturesAnalysis.RoleAnatomy[role]; ok {
			fmt.Fprintf(&b, "- Seen in %d features\n", anatomy.FeatureCount)
			if len(anatomy.BaseClasses) > 0 {
				fmt.Fprintf(&b, "- Common base: %s\n", anatomy.BaseClasses[0].Name)
			}
		} else {
			fmt.Fprintln(&b, "- No role anatomy found.")
		}
	}
	if featureName != "" {
		fmt.Fprintln(&b)
		if feature, ok := mapgen.FindFeature(conv, featureName); ok {
			fmt.Fprintf(&b, "Feature: %s\n", feature.Path)
			fmt.Fprintf(&b, "- Structure: %s\n", valueOrUnknown(feature.Structure))
			fmt.Fprintf(&b, "- Roles: %s\n", strings.Join(sortedStringMapKeys(feature.Files), ", "))
		} else {
			fmt.Fprintf(&b, "Feature %q was not found in scanned conventions.\n", featureName)
		}
	}
	return b.String()
}

func buildExplainAI(conv models.Convention, featureName, role string) string {
	conventions := output.BulletList([]string{
		"Project type: " + valueOrUnknown(conv.ProjectType),
		"Feature root: " + valueOrUnknown(conv.FeatureRoot),
		"State management: " + valueOrUnknown(conv.StateManagement),
		"Routing: " + valueOrUnknown(conv.Routing.Type),
		"Recommended pattern: " + recommendedPatternForExplain(conv),
	})
	findings := output.BulletList([]string{
		fmt.Sprintf("Features indexed: %d", len(conv.FeaturesAnalysis.Features)),
		fmt.Sprintf("Role anatomy entries: %d", len(conv.FeaturesAnalysis.RoleAnatomy)),
	})
	examples := output.BulletList(firstFeatureNames(conv, 5))
	var warnings []string
	if featureName != "" {
		if feature, ok := mapgen.FindFeature(conv, featureName); ok {
			findings += "\n" + output.BulletList([]string{"Selected feature: " + feature.Path})
		} else {
			warnings = append(warnings, "Feature "+featureName+" not found in scanned conventions.")
		}
	}
	if role != "" {
		if _, ok := conv.FeaturesAnalysis.RoleAnatomy[role]; ok {
			findings += "\n" + output.BulletList([]string{"Selected role: " + role})
		} else {
			warnings = append(warnings, "Role "+role+" not found in scanned anatomy.")
		}
	}
	warnings = append(warnings, "Run repox scan if conventions look outdated.")
	return output.Contract(
		"Repox convention explanation.",
		conventions,
		findings,
		examples,
		[]string{"repox map --ai", "repox plan feature <name> --ai", "repox generate feature <name> --like <existing> --dry-run"},
		warnings,
	)
}

func firstFeatureNames(conv models.Convention, limit int) []string {
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

func sortedStringMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func recommendedPatternForExplain(conv models.Convention) string {
	if conv.FeaturesAnalysis.RecommendedPattern != "" {
		return conv.FeaturesAnalysis.RecommendedPattern
	}
	return valueOrUnknown(conv.FeatureStructure)
}
