package models

import "time"

// Generation records a scaffold generation event.
type Generation struct {
	ID          string    `json:"id"`
	FeatureName string    `json:"feature_name"`
	Template    string    `json:"template"`
	Mode        string    `json:"mode"`         // "template" or "ai"
	Files       []string  `json:"files"`
	SnapshotDir string    `json:"snapshot_dir"` // set by repox learn (v0.5.0)
	CreatedAt   time.Time `json:"created_at"`
}
