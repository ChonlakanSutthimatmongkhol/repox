package models

// Example stores a real code example found in the repo for a given feature.
type Example struct {
	FeatureName string `json:"feature_name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
}
