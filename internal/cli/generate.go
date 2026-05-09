package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/retriever"
)

var (
	generateForce        bool
	generateDryRun       bool
	generateTemplate     string
	generateWithExamples bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate scaffolds for features",
}

var generateFeatureCmd = &cobra.Command{
	Use:   "feature <name>",
	Short: "Generate a feature scaffold",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateFeature,
}

func init() {
	generateFeatureCmd.Flags().BoolVarP(&generateForce, "force", "f", false, "Overwrite existing files")
	generateFeatureCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Preview files without writing")
	generateFeatureCmd.Flags().StringVarP(&generateTemplate, "template", "t", "", "Template to use (overrides config)")
	generateFeatureCmd.Flags().BoolVar(&generateWithExamples, "with-examples", false, "Find and show similar existing features before generating")
	generateCmd.AddCommand(generateFeatureCmd)
	rootCmd.AddCommand(generateCmd)
}

func runGenerateFeature(cmd *cobra.Command, args []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	featureName := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("generate: getwd: %w", err)
	}

	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	if err != nil {
		return fmt.Errorf("generate: load config: %w", err)
	}

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("generate: load conventions: %w", err)
	}

	tmplName := cfg.DefaultTemplate
	if generateTemplate != "" {
		tmplName = generateTemplate
	}

	if generateWithExamples {
		examples, err := config.Load[[]models.Example](config.RepoxPath("examples.json"))
		if err != nil || len(examples) == 0 {
			examples, _ = retriever.IndexFeatures(cwd, &conv)
		}
		similar := retriever.FindSimilar(featureName, examples, 3)
		if len(similar) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Similar features found:")
			for _, ex := range similar {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", ex.Name, ex.Path)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	gen := generator.NewTemplateGenerator()
	files, err := gen.Generate(featureName, tmplName, &conv)
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if generateDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run — no files written:")
		for _, f := range files {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Path)
		}
		return nil
	}

	results, err := generator.WriteFiles(files, cwd, generateForce)
	if err != nil {
		return fmt.Errorf("generate: write files: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	written, skipped := 0, 0
	for _, r := range results {
		if r.Written {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", green("created"), r.Path)
			written++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", yellow("skipped"), r.Reason)
			skipped++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%d created, %d skipped\n", written, skipped)
	return nil
}
