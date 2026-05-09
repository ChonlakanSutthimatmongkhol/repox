package models

// Lesson is a reusable convention extracted from developer edits to generated code.
type Lesson struct {
	ID         string       `json:"id"`
	Scope      string       `json:"scope"`      // template name or "global"
	Lesson     string       `json:"lesson"`     // human-readable lesson text
	Confidence float64      `json:"confidence"` // 0.0 – 1.0
	Approved   bool         `json:"approved"`
	Source     LessonSource `json:"source"`
}

// LessonSource records where a lesson was extracted from.
type LessonSource struct {
	GenerationID string `json:"generation_id"`
	Feature      string `json:"feature"`
	DetectedFrom string `json:"detected_from"` // "git_diff" or "manual"
}
