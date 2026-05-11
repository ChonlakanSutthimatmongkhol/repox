package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/mapgen"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/output"
)

var (
	mapFeature string
	mapFormat  string
	mapRender  bool
	mapOpen    bool
	mapAI      bool
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Generate project and convention maps",
	RunE:  runMap,
}

func init() {
	mapCmd.Flags().StringVar(&mapFeature, "feature", "", "Generate a map for a scanned feature")
	mapCmd.Flags().StringVar(&mapFormat, "format", "markdown", "Map format to emphasize (markdown or markmap)")
	mapCmd.Flags().BoolVar(&mapRender, "render", false, "Render markmap markdown to HTML when markmap is installed")
	mapCmd.Flags().BoolVar(&mapOpen, "open", false, "Open rendered map HTML when possible")
	mapCmd.Flags().BoolVar(&mapAI, "ai", false, "Print compact AI-friendly markdown")
	rootCmd.AddCommand(mapCmd)
}

func runMap(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox setup` first")
	}
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("map: load conventions: run `repox scan` first: %w", err)
	}
	files, warnings, err := generateMapFiles(conv, mapFeature, mapRender || mapOpen)
	if err != nil {
		return err
	}
	if mapOpen {
		opened, openWarning := openFirstHTML(files)
		if openWarning != "" {
			warnings = append(warnings, openWarning)
		}
		if opened != "" {
			warnings = append(warnings, "Opened "+opened)
		}
	}
	if mapAI {
		fmt.Fprint(cmd.OutOrStdout(), buildMapAI(conv, files, warnings))
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Generated Repox maps:")
	for _, file := range files {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", file)
	}
	for _, warning := range warnings {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s\n", warning)
	}
	return nil
}

func generateMapFiles(conv models.Convention, feature string, render bool) ([]string, []string, error) {
	maps := []mapgen.GeneratedMap{
		{Path: config.RepoxPath(filepath.Join("maps", "project.md")), Content: mapgen.GenerateProjectMarkdown(conv)},
		{Path: config.RepoxPath(filepath.Join("maps", "conventions.md")), Content: mapgen.GenerateConventionsMarkdown(conv)},
		{Path: config.RepoxPath(filepath.Join("maps", "project-markmap.md")), Content: mapgen.GenerateProjectMarkmap(conv)},
	}
	if feature != "" {
		if name, content, ok := mapgen.GenerateFeatureMarkdown(conv, feature); ok {
			maps = append(maps, mapgen.GeneratedMap{Path: config.RepoxPath(filepath.Join("maps", "features", name+".md")), Content: content})
		} else {
			return nil, nil, fmt.Errorf("map: feature %q not found in scanned conventions", feature)
		}
		if name, content, ok := mapgen.GenerateFeatureMarkmap(conv, feature); ok {
			maps = append(maps, mapgen.GeneratedMap{Path: config.RepoxPath(filepath.Join("maps", "features", name+"-markmap.md")), Content: content})
		}
	}
	var files []string
	for _, item := range maps {
		if err := os.MkdirAll(filepath.Dir(item.Path), 0o755); err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(item.Path, []byte(item.Content), 0o644); err != nil {
			return nil, nil, err
		}
		files = append(files, item.Path)
	}
	var warnings []string
	if render {
		rendered, warning := renderMarkmapFiles(files)
		files = append(files, rendered...)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	return files, warnings, nil
}

func renderMarkmapFiles(files []string) ([]string, string) {
	if markmapCommand() == "" {
		return nil, "Markmap renderer not found. Install with: npm install -g markmap-cli"
	}
	var rendered []string
	for _, file := range files {
		if filepath.Ext(file) != ".md" || !strings.Contains(filepath.Base(file), "markmap") {
			continue
		}
		out := strings.TrimSuffix(file, ".md") + ".html"
		cmd := markmapRenderCommand(file, out)
		if err := cmd.Run(); err != nil {
			return rendered, "Markmap render failed for " + file
		}
		rendered = append(rendered, out)
	}
	return rendered, ""
}

func markmapRenderCommand(input, output string) *exec.Cmd {
	for _, name := range []string{"markmap", "markmap-cli"} {
		if path, err := exec.LookPath(name); err == nil {
			return exec.Command(path, input, "-o", output)
		}
	}
	if path, err := exec.LookPath("npx"); err == nil {
		return exec.Command(path, "markmap-cli", input, "-o", output)
	}
	return exec.Command("markmap", input, "-o", output)
}

func openFirstHTML(files []string) (string, string) {
	for _, file := range files {
		if filepath.Ext(file) != ".html" {
			continue
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", file)
		case "linux":
			cmd = exec.Command("xdg-open", file)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", file)
		default:
			return "", "Open is unsupported; map HTML is at " + file
		}
		if err := cmd.Start(); err != nil {
			return "", "Could not open map; map HTML is at " + file
		}
		return file, ""
	}
	return "", "No rendered HTML found. Run repox map --render"
}

func buildMapAI(conv models.Convention, files []string, warnings []string) string {
	return output.Contract(
		"Generated Repox project maps.",
		output.BulletList([]string{
			"Project type: " + valueOrUnknown(conv.ProjectType),
			"Feature root: " + valueOrUnknown(conv.FeatureRoot),
			"Recommended pattern: " + recommendedPatternForExplain(conv),
		}),
		output.BulletList([]string{fmt.Sprintf("Features indexed: %d", len(conv.FeaturesAnalysis.Features))}),
		output.BulletList(firstFeatureNames(conv, 5)),
		[]string{"repox map --render", "repox explain --ai", "repox generate feature <name> --like <existing> --dry-run"},
		append([]string{"Run repox scan if maps look outdated."}, warnings...),
	) + output.Section("Generated Files", output.BulletList(files))
}
