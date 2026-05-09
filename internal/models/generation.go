package models

import "time"

// Generation records a scaffold generation event.
type Generation struct {
	ID          string    `json:"id"`
	FeatureName string    `json:"feature_name"`
	Template    string    `json:"template"`
	Files       []string  `json:"files"`
	CreatedAt   time.Time `json:"created_at"`
}
