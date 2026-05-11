package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose whether this repo is ready for Repox",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	fmt.Fprint(cmd.OutOrStdout(), buildDoctorReport())
	return nil
}

func buildDoctorReport() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Repox Doctor")
	fmt.Fprintln(&b)
	if !config.RepoxDirExists() {
		fmt.Fprintln(&b, "WARN .repox missing")
		fmt.Fprintln(&b, "     Fix: repox setup")
		return b.String()
	}
	fmt.Fprintln(&b, "OK   .repox exists")
	checkPath(&b, config.RepoxPath("config.json"), "config.json", "repox init --force")
	checkPath(&b, config.RepoxPath("conventions.json"), "conventions.json", "repox scan")

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err == nil {
		if conv.ProjectType != "" {
			fmt.Fprintf(&b, "OK   Project type: %s\n", conv.ProjectType)
		} else {
			fmt.Fprintln(&b, "WARN Project type missing")
			fmt.Fprintln(&b, "     Fix: repox scan")
		}
		if conv.FeatureRoot != "" {
			if _, statErr := os.Stat(conv.FeatureRoot); statErr == nil {
				fmt.Fprintf(&b, "OK   Feature root: %s\n", conv.FeatureRoot)
			} else {
				fmt.Fprintf(&b, "WARN Feature root missing on disk: %s\n", conv.FeatureRoot)
				fmt.Fprintln(&b, "     Fix: repox scan --feature-root <path>")
			}
		}
		fmt.Fprintf(&b, "OK   Features indexed: %d\n", len(conv.FeaturesAnalysis.Features))
	} else {
		fmt.Fprintln(&b, "WARN Conventions unreadable")
		fmt.Fprintln(&b, "     Fix: repox scan")
	}
	checkPath(&b, config.RepoxPath(filepath.Join("skill", "SKILL.md")), "skill/SKILL.md", "repox skill generate")
	if _, err := os.Stat(config.RepoxPath(filepath.Join("maps", "project.md"))); err == nil {
		fmt.Fprintln(&b, "OK   Map generated")
	} else {
		fmt.Fprintln(&b, "WARN Map not generated")
		fmt.Fprintln(&b, "     Fix: repox map")
	}
	if markmapCommand() != "" {
		fmt.Fprintln(&b, "OK   Markmap renderer available")
	} else {
		fmt.Fprintln(&b, "WARN Markmap renderer missing")
		fmt.Fprintln(&b, "     Fix: npm install -g markmap-cli")
	}
	return b.String()
}

func checkPath(b *strings.Builder, path, label, fix string) {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(b, "OK   %s exists\n", label)
		return
	}
	fmt.Fprintf(b, "WARN %s missing\n", label)
	fmt.Fprintf(b, "     Fix: %s\n", fix)
}

func markmapCommand() string {
	for _, name := range []string{"markmap", "markmap-cli"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if path, err := exec.LookPath("npx"); err == nil {
		return path + " markmap-cli"
	}
	return ""
}
