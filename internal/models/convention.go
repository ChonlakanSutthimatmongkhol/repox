// Package models contains shared data types used across repox.
package models

// Convention holds the detected or configured conventions for a project.
type Convention struct {
	ProjectType       string            `json:"project_type"`
	StateManagement   string            `json:"state_management"`
	FeatureStructure  string            `json:"feature_structure"`
	FeatureRoot       string            `json:"feature_root"`
	TestRoot          string            `json:"test_root"`
	Naming            NamingConvention  `json:"naming"`
	Routing           RoutingConfig     `json:"routing"`
	CommonImports     []string          `json:"common_imports"`
	FeaturesAnalysis  FeaturesAnalysis  `json:"features_analysis"`
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

// FeatureInfo holds metadata about a detected feature.
type FeatureInfo struct {
	Name         string `json:"name"`
	Structure    string `json:"structure"`
	LastModified string `json:"last_modified"`
	Path         string `json:"path"`
}

// PatternMapping defines how to route a file type to subdirectories for a pattern.
type PatternMapping struct {
	FileName string `json:"file_name"`
	Subdir   string `json:"subdir"`
}

// FeaturesAnalysis holds analysis results of all features in the project.
type FeaturesAnalysis struct {
	TotalFeatures       int                         `json:"total_features"`
	PatternDistribution map[string]int              `json:"pattern_distribution"`
	RecommendedPattern  string                      `json:"recommended_pattern"`
	LatestPattern       string                      `json:"latest_pattern"`
	Features            []FeatureInfo               `json:"features"`
	PatternMappings     map[string][]PatternMapping `json:"pattern_mappings"`
}
