// Package output contains small formatting helpers shared by AI-friendly CLI output.
package output

import (
	"fmt"
	"strings"
)

// Section renders a markdown section. Empty bodies render as "None." so the AI
// contract stays predictable for downstream agents.
func Section(title string, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		body = "None."
	}
	return fmt.Sprintf("## %s\n%s\n", title, body)
}

// BulletList renders markdown bullets from non-empty items.
func BulletList(items []string) string {
	var b strings.Builder
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return strings.TrimSpace(b.String())
}

// SuggestedCommands renders command bullets for the standard AI contract.
func SuggestedCommands(commands []string) string {
	return BulletList(commands)
}

// Contract joins the standard Repox AI output sections.
func Contract(summary, conventions, findings, examples string, commands []string, warnings []string) string {
	sections := []string{
		Section("Summary", summary),
		Section("Detected Conventions", conventions),
		Section("Important Findings", findings),
		Section("Related Examples", examples),
		Section("Suggested Next Commands", SuggestedCommands(commands)),
		Section("Warnings", BulletList(warnings)),
	}
	return strings.TrimSpace(strings.Join(sections, "\n")) + "\n"
}
