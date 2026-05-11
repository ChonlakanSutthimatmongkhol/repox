package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Initialize, scan, and generate Repox AI instructions",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		initForce = false
		if err := runInit(cmd, nil); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "OK   Initialized .repox/")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "OK   .repox already initialized")
	}
	oldScanAI := scanAI
	scanAI = false
	defer func() { scanAI = oldScanAI }()
	if err := runScan(cmd, nil); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "OK   Scanned project conventions")

	oldCopilot := skillWriteCopilot
	skillWriteCopilot = false
	defer func() { skillWriteCopilot = oldCopilot }()
	if err := runSkillGenerate(cmd, nil); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "OK   Generated .repox/skill/SKILL.md")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Next:")
	fmt.Fprintln(cmd.OutOrStdout(), "  repox doctor")
	fmt.Fprintln(cmd.OutOrStdout(), "  repox map --open")
	fmt.Fprintln(cmd.OutOrStdout(), "  repox explain --ai")
	fmt.Fprintln(cmd.OutOrStdout(), "  repox new feature <name> --like <existing-feature> --preview")
	return nil
}
