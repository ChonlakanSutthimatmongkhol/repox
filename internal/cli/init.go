package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize repox in the current directory",
	Long:  "Creates the .repox/ directory with default configuration and empty data files.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing .repox/ directory")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, _ []string) error {
	const repoxDir = ".repox"

	if config.RepoxDirExists() && !initForce {
		return fmt.Errorf(".repox/ already exists. Use --force to reinitialize")
	}

	if err := os.MkdirAll(repoxDir, 0o755); err != nil {
		return fmt.Errorf("init: create %s: %w", repoxDir, err)
	}

	files := map[string]any{
		"config.json":      config.DefaultConfig(),
		"conventions.json": config.DefaultConventions(),
		"examples.json":    []models.Example{},
		"lessons.json":     []models.Lesson{},
		"generations.json": []models.Generation{},
	}

	for name, v := range files {
		path := config.RepoxPath(name)
		if err := config.Save(path, v); err != nil {
			return fmt.Errorf("init: write %s: %w", name, err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Initialized repox in .repox/")
	return nil
}
