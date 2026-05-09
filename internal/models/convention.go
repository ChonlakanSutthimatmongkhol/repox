// Package models contains shared data types used across repox.
package models

// Convention holds the detected or configured conventions for a project.
type Convention struct {
	ProjectType      string           `json:"project_type"`
	StateManagement  string           `json:"state_management"`
	FeatureStructure string           `json:"feature_structure"`
	FeatureRoot      string           `json:"feature_root"`
	TestRoot         string           `json:"test_root"`
	Naming           NamingConvention `json:"naming"`
	Routing          RoutingConfig    `json:"routing"`
	CommonImports    []string         `json:"common_imports"`
}

// NamingConvention holds the naming suffix and case rules for generated files.
type NamingConvention struct {
	ClassCase        string `json:"class_case"`
	FileCase         string `json:"file_case"`
	ScreenSuffix     string `json:"screen_suffix"`
	BlocSuffix       string `json:"bloc_suffix"`
	EventSuffix      string `json:"event_suffix"`
	StateSuffix      string `json:"state_suffix"`
	RepositorySuffix string `json:"repository_suffix"`
	UsecaseSuffix    string `json:"usecase_suffix"`
}

// RoutingConfig holds routing configuration.
type RoutingConfig struct {
	Type      string `json:"type"`
	RouteFile string `json:"route_file"`
}
