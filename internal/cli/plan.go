package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var (
	planRoles string
	planLike  string
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview feature generation plans",
}

var planFeatureCmd = &cobra.Command{
	Use:   "feature <name-or-path>",
	Short: "Explain what Repox would generate for a feature",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanFeature,
}

func init() {
	planFeatureCmd.Flags().StringVar(&planRoles, "roles", "", "Comma-separated file roles to include in the plan")
	planFeatureCmd.Flags().StringVar(&planLike, "like", "", "Use an existing feature path/name as the shape reference")
	planCmd.AddCommand(planFeatureCmd)
	rootCmd.AddCommand(planCmd)
}

func runPlanFeature(cmd *cobra.Command, args []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	featureName := args[0]
	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	if err != nil {
		return fmt.Errorf("plan: load config: %w", err)
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("plan: load conventions: %w", err)
	}
	applyRecommendedPattern(&conv)

	roles := parseRoles(planRoles)
	if err := applyGenerateLike(&conv, featureName, planLike, &roles); err != nil {
		return err
	}

	gen := generator.NewTemplateGenerator()
	files, err := gen.GenerateWithOptions(featureName, cfg.DefaultTemplate, &conv, generator.GenerateOptions{Roles: roles})
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}

	printFeaturePlan(cmd, featureName, planLike, conv, roles, files)
	return nil
}

func printFeaturePlan(cmd *cobra.Command, featureName, like string, conv models.Convention, roles []string, files []generator.GeneratedFile) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Feature plan: %s\n", featureName)
	fmt.Fprintf(out, "  Pattern: %s\n", conv.FeatureStructure)
	if like != "" {
		fmt.Fprintf(out, "  Like:    %s\n", like)
	}
	if len(roles) > 0 {
		fmt.Fprintf(out, "  Roles:   %s\n", strings.Join(roles, ", "))
	} else {
		fmt.Fprintln(out, "  Roles:   all template roles")
	}

	fmt.Fprintln(out, "\nFiles:")
	for _, file := range files {
		fmt.Fprintf(out, "  - %s\n", file.Path)
	}

	roleAnatomy := selectedRoleAnatomy(conv.FeaturesAnalysis.RoleAnatomy, roles)
	if len(roleAnatomy) == 0 {
		return
	}
	fmt.Fprintln(out, "\nRole anatomy hints:")
	roleNames := make([]string, 0, len(roleAnatomy))
	for role := range roleAnatomy {
		roleNames = append(roleNames, role)
	}
	sort.Strings(roleNames)
	for _, role := range roleNames {
		anatomy := roleAnatomy[role]
		fmt.Fprintf(out, "  - %s", role)
		details := []string{}
		if len(anatomy.BaseClasses) > 0 {
			details = append(details, "base "+anatomy.BaseClasses[0].Name)
		}
		if len(anatomy.Methods) > 0 {
			details = append(details, "methods "+strings.Join(anatomyVoteNamesForPlan(anatomy.Methods, 3), ", "))
		}
		if len(anatomy.Capabilities) > 0 {
			details = append(details, "capabilities "+strings.Join(anatomyVoteNamesForPlan(anatomy.Capabilities, 3), ", "))
		}
		if len(details) > 0 {
			fmt.Fprintf(out, ": %s", strings.Join(details, "; "))
		}
		fmt.Fprintln(out)
	}
}

func selectedRoleAnatomy(roleAnatomy map[string]models.RoleAnatomy, roles []string) map[string]models.RoleAnatomy {
	if len(roleAnatomy) == 0 {
		return nil
	}
	if len(roles) == 0 {
		return roleAnatomy
	}
	selected := map[string]models.RoleAnatomy{}
	for _, role := range roles {
		if anatomy, ok := roleAnatomy[role]; ok {
			selected[role] = anatomy
		}
	}
	return selected
}

func anatomyVoteNamesForPlan(votes []models.AnatomyVote, limit int) []string {
	if len(votes) < limit {
		limit = len(votes)
	}
	names := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		names = append(names, votes[i].Name)
	}
	return names
}
