package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/templatex"
)

var (
	templateName string
	templateFrom string
)

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage Repox templates",
}

var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List extracted project templates",
	RunE:  runTemplateList,
}

var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a template from an existing feature",
	RunE:  runTemplateCreate,
}

func init() {
	templateCreateCmd.Flags().StringVar(&templateName, "name", "", "Template name")
	templateCreateCmd.Flags().StringVar(&templateFrom, "from", "", "Existing feature to extract")
	templateCmd.AddCommand(templateListCmd, templateCreateCmd)
	rootCmd.AddCommand(templateCmd)
}

func runTemplateList(cmd *cobra.Command, _ []string) error {
	root := config.RepoxPath("templates")
	entries, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "No extracted templates found.")
		return nil
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No extracted templates found.")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Extracted templates:")
	for _, name := range names {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", name)
	}
	return nil
}

func runTemplateCreate(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox setup` first")
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("template: load conventions: run `repox scan` first: %w", err)
	}
	result, err := templatex.Extract(conv, templatex.ExtractOptions{Name: templateName, From: templateFrom, Root: "."})
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Created template %s\n", filepath.ToSlash(result.Dir))
	for _, file := range result.Files {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", filepath.ToSlash(file))
	}
	return nil
}
