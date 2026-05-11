package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/output"
)

var (
	planRoles string
	planLike  string
	planAI    bool
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
	planFeatureCmd.Flags().BoolVar(&planAI, "ai", false, "Print compact AI-friendly markdown")
	planCmd.AddCommand(planFeatureCmd)
	rootCmd.AddCommand(planCmd)
}

func runPlanFeature(cmd *cobra.Command, args []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	featureName := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("plan: getwd: %w", err)
	}
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
	var likeFeature *models.FeatureAnalysis
	if strings.TrimSpace(planLike) != "" {
		feature, ok := findFeatureAnalysis(&conv, planLike)
		if ok {
			likeFeature = &feature
		}
	}
	if err := applyGenerateLike(&conv, featureName, planLike, cfg.DefaultTemplate, &roles); err != nil {
		return err
	}

	gen := generator.NewTemplateGenerator()
	files, err := gen.GenerateWithOptions(featureName, cfg.DefaultTemplate, &conv, generator.GenerateOptions{
		Roles:         roles,
		RolesExplicit: strings.TrimSpace(planRoles) != "",
		LikeFeature:   likeFeature,
		BaseDir:       cwd,
	})
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	if strings.TrimSpace(planRoles) == "" {
		roles = rolesFromPlannedFiles(featureName, roles, files)
	}

	if planAI {
		fmt.Fprint(cmd.OutOrStdout(), buildPlanAI(featureName, planLike, conv, roles, files))
		return nil
	}
	printFeaturePlan(cmd, featureName, planLike, conv, roles, files)
	return nil
}

func buildPlanAI(featureName, like string, conv models.Convention, roles []string, files []generator.GeneratedFile) string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	roleText := "all template roles"
	if len(roles) > 0 {
		roleText = strings.Join(roles, ", ")
	}
	warnings := []string{}
	if strings.TrimSpace(like) == "" {
		warnings = append(warnings, "No --like provided. Prefer using a similar existing feature.")
	}
	return output.Contract(
		fmt.Sprintf("Plan for feature `%s`.", featureName),
		output.BulletList([]string{
			"Pattern: " + valueOrUnknown(conv.FeatureStructure),
			"Suggested roles: " + roleText,
		}),
		"Files planned:\n"+output.BulletList(paths),
		output.BulletList(firstFeatureNames(conv, 5)),
		[]string{"repox generate feature " + featureName + likeCommandPart(like) + " --dry-run", "repox map --feature " + featureName + " --ai"},
		warnings,
	)
}

func rolesFromPlannedFiles(featureName string, roles []string, files []generator.GeneratedFile) []string {
	seen := map[string]bool{}
	for _, role := range roles {
		if role == "" {
			continue
		}
		seen[role] = true
	}
	featurePath := normalizeFeaturePathForCLI(featureName)
	featureSnake := generator.ToSnakeCase(featurePath[strings.LastIndex(featurePath, string('/'))+1:])
	for _, file := range files {
		base := strings.TrimSuffix(file.Path[strings.LastIndex(file.Path, "/")+1:], ".dart")
		base = strings.TrimSuffix(base, ".go")
		if !strings.HasPrefix(base, featureSnake+"_") {
			continue
		}
		role := strings.TrimPrefix(base, featureSnake+"_")
		if role != "" {
			seen[role] = true
		}
	}
	merged := make([]string, 0, len(seen))
	for role := range seen {
		merged = append(merged, role)
	}
	sort.Strings(merged)
	return merged
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
