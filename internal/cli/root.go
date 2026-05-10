// Package cli contains all cobra command definitions.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/mcp"
)

const version = "1.0.13"

var mcpMode bool

var rootCmd = &cobra.Command{
	Use:   "repox",
	Short: "Generate feature scaffolds matching your repo's conventions",
	Long:  `Repox scans a codebase, learns project conventions, and generates feature scaffolds matching the team's existing code style.`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		if mcpMode {
			return mcp.Serve()
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("repox v{{.Version}}\n")
	rootCmd.PersistentFlags().BoolVar(&mcpMode, "mcp", false, "Start as MCP server on stdio")
}
