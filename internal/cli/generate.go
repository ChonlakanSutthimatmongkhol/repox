package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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

	// --with-examples: show similar features found
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

	// Generate files from local templates.
	genMode := "template"
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
	var writtenPaths []string
	for _, r := range results {
		if r.Written {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", green("created"), r.Path)
			writtenPaths = append(writtenPaths, filepath.Join(cwd, r.Path))
			written++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", yellow("skipped"), r.Reason)
			skipped++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d created, %d skipped\n", written, skipped)

	// Run formatter on written files
	runFormatter(writtenPaths, cmd)

	// Log generation (and save snapshot for learner)
	genID := fmt.Sprintf("gen_%d", time.Now().Unix())
	filePaths := make([]string, len(results))
	for i, r := range results {
		filePaths[i] = r.Path
	}

	snapshotDir := saveSnapshot(genID, files, cwd)

	_ = appendGeneration(models.Generation{
		ID:          genID,
		FeatureName: featureName,
		Template:    tmplName,
		Mode:        genMode,
		Files:       filePaths,
		SnapshotDir: snapshotDir,
		CreatedAt:   time.Now(),
	})

	return nil
}

// saveSnapshot copies generated file contents to .repox/snapshots/<genID>/ for later diff.
func saveSnapshot(genID string, files []generator.GeneratedFile, baseDir string) string {
	snapDir := config.RepoxPath("snapshots/" + genID)
	for _, f := range files {
		dest := filepath.Join(snapDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}
		_ = os.WriteFile(dest, []byte(f.Content), 0o644)
	}
	// Return absolute path so diff_reader can locate it from any working directory.
	abs, err := filepath.Abs(snapDir)
	if err != nil {
		return snapDir
	}
	return abs
}

func appendGeneration(gen models.Generation) error {
	existing, err := config.Load[[]models.Generation](config.RepoxPath("generations.json"))
	if err != nil {
		existing = []models.Generation{}
	}
	existing = append(existing, gen)
	return config.Save(config.RepoxPath("generations.json"), existing)
}

func runFormatter(paths []string, cmd *cobra.Command) {
	if len(paths) == 0 {
		return
	}
	dartPath, err := exec.LookPath("dart")
	if err != nil {
		return // dart not installed — silently skip
	}

	var dartFiles []string
	for _, p := range paths {
		if filepath.Ext(p) == ".dart" {
			dartFiles = append(dartFiles, p)
		}
	}
	if len(dartFiles) == 0 {
		return
	}

	args := append([]string{"format"}, dartFiles...)
	out, err := exec.Command(dartPath, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: dart format failed: %s\n", string(out))
	}
}
