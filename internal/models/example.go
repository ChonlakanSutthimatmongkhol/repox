package models

// Example represents an indexed existing feature in the repository.
type Example struct {
	Name     string            `json:"name"`
	Path     string            `json:"path"`
	Files    map[string]string `json:"files"`    // role → relative path
	Patterns []string          `json:"patterns"` // detected patterns (e.g. "uses flutter_bloc")
	Metadata FeatureMetadata   `json:"metadata"`
}

// FeatureMetadata describes which components a feature contains.
type FeatureMetadata struct {
	HasBloc       bool     `json:"has_bloc"`
	HasScreen     bool     `json:"has_screen"`
	HasRepository bool     `json:"has_repository"`
	HasUseCase    bool     `json:"has_usecase"`
	HasHandler    bool     `json:"has_handler"`
	HasService    bool     `json:"has_service"`
	HasTest       bool     `json:"has_test"`
	Imports       []string `json:"imports"`
	Structure     string   `json:"structure"` // clean_architecture, grouped, flat
}
