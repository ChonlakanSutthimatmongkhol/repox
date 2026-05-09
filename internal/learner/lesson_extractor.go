package learner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/ai"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

const extractionSystemPrompt = `You are analyzing how a developer changed AI-generated scaffold code.
Identify reusable lessons for future scaffolding.
Return ONLY valid JSON (no markdown fences).`

type lessonCandidate struct {
	Scope      string  `json:"scope"`
	Lesson     string  `json:"lesson"`
	Confidence float64 `json:"confidence"`
}

type lessonExtractionResponse struct {
	Lessons []lessonCandidate `json:"lessons"`
}

// ExtractLessons sends changed diffs to the AI and returns candidate lessons.
func ExtractLessons(diffs []DiffResult, generationID, featureName, scope string, caller ai.Caller) ([]models.Lesson, error) {
	changed := make([]DiffResult, 0, len(diffs))
	for _, d := range diffs {
		if d.Changed {
			changed = append(changed, d)
		}
	}
	if len(changed) == 0 {
		return nil, nil
	}

	userPrompt := buildExtractionPrompt(changed, scope)
	rawText, err := caller.Call(extractionSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("learner: extract lessons: %w", err)
	}

	extracted, err := ParseExtractionJSON(rawText)
	if err != nil {
		return nil, err
	}

	var lessons []models.Lesson
	for i, c := range extracted.Lessons {
		if c.Lesson == "" {
			continue
		}
		s := c.Scope
		if s == "" {
			s = scope
		}
		lessons = append(lessons, models.Lesson{
			ID:         fmt.Sprintf("lesson_%s_%d", generationID, i),
			Scope:      s,
			Lesson:     c.Lesson,
			Confidence: c.Confidence,
			Approved:   false,
			Source: models.LessonSource{
				GenerationID: generationID,
				Feature:      featureName,
				DetectedFrom: "git_diff",
			},
		})
	}
	return lessons, nil
}

func buildExtractionPrompt(diffs []DiffResult, scope string) string {
	var b strings.Builder
	for _, d := range diffs {
		fmt.Fprintf(&b, "File: %s\n\nGenerated code:\n%s\n\nFinal code:\n%s\n\nDiff:\n%s\n\n---\n\n",
			d.Path, truncate(d.GeneratedCode, 80), truncate(d.CurrentCode, 80), truncate(d.Diff, 120))
	}

	fmt.Fprintf(&b, `Identify reusable lessons for scope "%s".
Focus on: widget/class substitutions, import corrections, naming patterns, architecture preferences, code style.
Skip: business logic specific to this feature, one-time typos.

Return ONLY valid JSON:
{"lessons":[{"scope":"%s","lesson":"...","confidence":0.8}]}`, scope, scope)

	return b.String()
}

// ParseExtractionJSON parses a raw AI response into lesson candidates.
func ParseExtractionJSON(raw string) (*lessonExtractionResponse, error) {
	s := strings.TrimSpace(raw)
	for _, fence := range []string{"```json", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			if idx := strings.LastIndex(s, "```"); idx != -1 {
				s = s[:idx]
			}
			s = strings.TrimSpace(s)
			break
		}
	}

	var resp lessonExtractionResponse
	if err := json.Unmarshal([]byte(s), &resp); err != nil {
		return nil, fmt.Errorf("learner: parse lessons JSON: %w\nraw: %.300s", err, raw)
	}
	return &resp, nil
}

func truncate(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n") + "\n... (truncated)"
}
