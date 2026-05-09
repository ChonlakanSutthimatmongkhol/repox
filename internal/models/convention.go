// Package models contains shared data types used across repox.
package models

// Convention holds the detected or configured conventions for a project.
type Convention struct {
	ProjectType      string           `json:"project_type"`
	StateManagement  string           `json:"state_management"`
	FeatureStructure string           `json:"feature_structure"`
	FeatureRoot      string           `json:"feature_root"`
	TestRoot         string           `json:"test_root"`
	ModulePath       string           `json:"module_path,omitempty"`
	Naming           NamingConvention `json:"naming"`
	Routing          RoutingConfig    `json:"routing"`
	CommonImports    []string         `json:"common_imports"`
	FeaturesAnalysis FeaturesAnalysis `json:"features_analysis,omitempty"`
	PatternMappings  PatternMappings  `json:"pattern_mappings,omitempty"`
}

// NamingConvention holds the naming suffix and case rules for generated files.
type NamingConvention struct {
	ClassCase        string `json:"class_case"`
	FileCase         string `json:"file_case"`
	ScreenSuffix     string `json:"screen_suffix,omitempty"`
	BlocSuffix       string `json:"bloc_suffix,omitempty"`
	EventSuffix      string `json:"event_suffix,omitempty"`
	StateSuffix      string `json:"state_suffix,omitempty"`
	RepositorySuffix string `json:"repository_suffix,omitempty"`
	UsecaseSuffix    string `json:"usecase_suffix,omitempty"`
	HandlerSuffix    string `json:"handler_suffix,omitempty"`
	ServiceSuffix    string `json:"service_suffix,omitempty"`
}

// RoutingConfig holds routing configuration.
type RoutingConfig struct {
	Type      string `json:"type"`
	RouteFile string `json:"route_file"`
}

// FeaturesAnalysis captures the project-wide feature structure scan.
type FeaturesAnalysis struct {
	Features            []FeatureAnalysis              `json:"features"`
	PatternDistribution map[string]PatternDistribution `json:"pattern_distribution"`
	RecommendedPattern  string                         `json:"recommended_pattern"`
	LatestPattern       string                         `json:"latest_pattern"`
}

// FeatureAnalysis describes one feature folder under the detected feature root.
type FeatureAnalysis struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	Structure    string `json:"structure"`
	LastModified string `json:"last_modified"`
}

// PatternDistribution stores the count and percentage for a feature structure.
type PatternDistribution struct {
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// PatternMappings describes where each template kind should be routed per pattern.
type PatternMappings map[string]PatternMapping

// PatternMapping stores file routing for a supported feature structure.
type PatternMapping struct {
	FileRoutes map[string]string `json:"file_routes"`
}
