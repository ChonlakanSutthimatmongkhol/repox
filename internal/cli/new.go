package cli

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Friendly aliases for creating code",
}

var newFeatureCmd = &cobra.Command{
	Use:   "feature <name-or-path>",
	Short: "Create a feature scaffold",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateFeature,
}

func init() {
	newFeatureCmd.Flags().BoolVarP(&generateForce, "force", "f", false, "Overwrite existing files")
	newFeatureCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Preview files without writing")
	newFeatureCmd.Flags().BoolVar(&generatePreview, "preview", false, "Alias for --dry-run")
	newFeatureCmd.Flags().StringVarP(&generateTemplate, "template", "t", "", "Template to use (overrides config)")
	newFeatureCmd.Flags().StringVar(&generatePattern, "pattern", "", "Feature pattern to use (flat, grouped, clean_architecture)")
	newFeatureCmd.Flags().BoolVar(&generateWithExamples, "with-examples", false, "Find and show similar existing features before generating")
	newFeatureCmd.Flags().StringVar(&generateRoles, "roles", "", "Comma-separated file roles to generate (e.g. bloc,event,state,screen)")
	newFeatureCmd.Flags().StringVar(&generateLike, "like", "", "Use an existing feature path/name as the shape reference")
	newFeatureCmd.Flags().BoolVar(&generateAI, "ai", false, "Print compact AI-friendly markdown for dry runs")
	newCmd.AddCommand(newFeatureCmd)
	rootCmd.AddCommand(newCmd)
}
