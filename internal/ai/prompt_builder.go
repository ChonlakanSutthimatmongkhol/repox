package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

const maxTokenEstimate = 30_000 // chars / 4 ≈ tokens; cap at 30K tokens
const maxExampleFiles = 2
const maxExampleLines = 100

// BuildSystemPrompt returns the fixed system prompt for Repox.
func BuildSystemPrompt() string {
	return `You are Repox, an AI scaffold generator.
Generate code that strictly follows the existing repository conventions.

Rules:
- Follow the provided folder structure exactly.
- Follow naming conventions exactly.
- Use the provided examples as the source of truth for code style.
- Do not invent new architecture or patterns.
- Do not add unnecessary dependencies.
- Return ONLY valid JSON with no markdown fences.`
}

// BuildUserPrompt constructs the user-facing prompt from the request context.
func BuildUserPrompt(req GenerateRequest) string {
	var b strings.Builder

	b.WriteString("Generate a new feature scaffold.\n\n")
	fmt.Fprintf(&b, "Feature name: %s\n\n", req.FeatureName)

	// Conventions JSON
	convJSON, _ := json.MarshalIndent(req.Conventions, "", "  ")
	fmt.Fprintf(&b, "Project conventions:\n%s\n\n", convJSON)

	// Examples (with token budget)
	examplesSection := buildExamplesSection(req.Examples, req.RootDir)
	b.WriteString("Relevant examples:\n")
	b.WriteString(examplesSection)
	b.WriteString("\n")

	// Lessons — inject only approved lessons matching this scope (top 20 by confidence)
	approved := filterLessons(req.Lessons, req.TargetTemplate)
	if len(approved) > 0 {
		lessonsJSON, _ := json.MarshalIndent(approved, "", "  ")
		fmt.Fprintf(&b, "Lessons learned (apply these to your output):\n%s\n\n", lessonsJSON)
	}

	// Target files
	b.WriteString("Required output files:\n")
	for _, f := range req.TargetFiles {
		fmt.Fprintf(&b, "  - %s\n", f)
	}

	b.WriteString(`
Return ONLY valid JSON (no markdown fences):
{
  "files": [
    {"path": "...", "content": "..."}
  ]
}`)

	return b.String()
}

func buildExamplesSection(examples []models.Example, rootDir string) string {
	if len(examples) == 0 {
		return "(no examples available)\n"
	}

	var b strings.Builder
	totalChars := 0
	charLimit := maxTokenEstimate * 4 / 2 // reserve half budget for examples

	for _, ex := range examples {
		if totalChars > charLimit {
			break
		}
		fmt.Fprintf(&b, "### Feature: %s (%s)\n", ex.Name, ex.Path)

		count := 0
		for role, relPath := range ex.Files {
			if count >= maxExampleFiles {
				break
			}
			content := readTruncated(filepath.Join(rootDir, relPath), maxExampleLines)
			if content == "" {
				continue
			}
			fmt.Fprintf(&b, "// %s (%s)\n%s\n\n", relPath, role, content)
			totalChars += len(content)
			count++
		}
	}
	return b.String()
}

// filterLessons returns approved lessons matching the given template scope, sorted by confidence desc, max 20.
func filterLessons(lessons []models.Lesson, templateScope string) []models.Lesson {
	var matched []models.Lesson
	for _, l := range lessons {
		if !l.Approved {
			continue
		}
		if l.Scope == "global" || l.Scope == templateScope {
			matched = append(matched, l)
		}
	}
	// Sort by confidence descending
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[j].Confidence > matched[i].Confidence {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}
	if len(matched) > 20 {
		matched = matched[:20]
	}
	return matched
}

func readTruncated(path string, maxLines int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < maxLines {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}
