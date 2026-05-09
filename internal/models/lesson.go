package models

// Lesson is a pattern extracted from repo diffs or examples.
type Lesson struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Source      string `json:"source"`
}
