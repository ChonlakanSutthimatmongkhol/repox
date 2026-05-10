// Package models contains shared data types used across repox.
package models

// Convention holds the detected or configured conventions for a project.
type Convention struct {
	ProjectType      string                    `json:"project_type"`
	StateManagement  string                    `json:"state_management"`
	FeatureStructure string                    `json:"feature_structure"`
	FeatureRoot      string                    `json:"feature_root"`
	TestRoot         string                    `json:"test_root"`
	ModulePath       string                    `json:"module_path,omitempty"`
	Naming           NamingConvention          `json:"naming"`
	Routing          RoutingConfig             `json:"routing"`
	CommonImports    []string                  `json:"common_imports"`
	FeaturesAnalysis FeaturesAnalysis          `json:"features_analysis,omitempty"`
	PatternMappings  PatternMappings           `json:"pattern_mappings,omitempty"`
	Roles            map[string]RoleConvention `json:"roles,omitempty"`
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
	// SuffixRoles maps lowercase file suffixes (e.g. "screen", "bloc") to repox
	// role names (e.g. "screen", "bloc"). Built by the scanner from detected suffixes
	// so generators need no hardcoded suffix→role mapping.
	SuffixRoles map[string]string `json:"suffix_roles,omitempty"`
}

// RoutingConfig holds routing configuration.
type RoutingConfig struct {
	Type      string `json:"type"`
	RouteFile string `json:"route_file"`
}

// RoleConvention describes how one scanned project role is named on disk and in code.
type RoleConvention struct {
	FileSuffix  string `json:"file_suffix,omitempty"`
	ClassSuffix string `json:"class_suffix,omitempty"`
}

// FeaturesAnalysis captures the project-wide feature structure scan.
type FeaturesAnalysis struct {
	Features            []FeatureAnalysis              `json:"features"`
	PatternDistribution map[string]PatternDistribution `json:"pattern_distribution"`
	RecommendedPattern  string                         `json:"recommended_pattern"`
	LatestPattern       string                         `json:"latest_pattern"`
	RoleAnatomy         map[string]RoleAnatomy         `json:"role_anatomy,omitempty"`
}

// FeatureAnalysis describes one feature folder under the detected feature root.
type FeatureAnalysis struct {
	Name         string                 `json:"name"`
	Path         string                 `json:"path"`
	Parent       string                 `json:"parent,omitempty"`
	Depth        int                    `json:"depth,omitempty"`
	Structure    string                 `json:"structure"`
	LastModified string                 `json:"last_modified"`
	FileCount    int                    `json:"file_count,omitempty"`
	Files        map[string]string      `json:"files,omitempty"`
	FileRoutes   map[string]string      `json:"file_routes,omitempty"`
	Anatomy      map[string]FileAnatomy `json:"anatomy,omitempty"`
}

// FileAnatomy captures the structural shape of one role file.
type FileAnatomy struct {
	Role                string              `json:"role"`
	Path                string              `json:"path"`
	ClassNames          []string            `json:"class_names,omitempty"`
	Types               []TypeAnatomy       `json:"types,omitempty"`
	Functions           []FunctionSignature `json:"functions,omitempty"`
	BaseClasses         []string            `json:"base_classes,omitempty"`
	Mixins              []string            `json:"mixins,omitempty"`
	Methods             []string            `json:"methods,omitempty"`
	AbstractOverrides   []string            `json:"abstract_overrides,omitempty"` // "@override" public method signatures
	ConstructorDeps     []string            `json:"constructor_deps,omitempty"`
	Imports             []string            `json:"imports,omitempty"`
	Capabilities        []string            `json:"capabilities,omitempty"`
	HasFirebaseTracking bool                `json:"has_firebase_tracking,omitempty"`
}

// TypeAnatomy captures a scanned type declaration.
type TypeAnatomy struct {
	Name       string              `json:"name"`
	Kind       string              `json:"kind,omitempty"`
	Extends    string              `json:"extends,omitempty"`
	Implements []string            `json:"implements,omitempty"`
	Mixins     []string            `json:"mixins,omitempty"`
	Methods    []FunctionSignature `json:"methods,omitempty"`
}

// FunctionSignature captures a scanned function or method signature.
type FunctionSignature struct {
	Name       string      `json:"name"`
	Receiver   string      `json:"receiver,omitempty"`
	ReturnType string      `json:"return_type,omitempty"`
	Returns    string      `json:"returns,omitempty"`
	Params     []Parameter `json:"params,omitempty"`
	Signature  string      `json:"signature,omitempty"`
	IsMethod   bool        `json:"is_method,omitempty"`
	IsAsync    bool        `json:"is_async,omitempty"`
	IsOverride bool        `json:"is_override,omitempty"`
}

// Parameter captures a scanned function parameter.
type Parameter struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// RoleAnatomy summarizes common anatomy for a role across discovered features.
type RoleAnatomy struct {
	FeatureCount        int           `json:"feature_count"`
	BaseClasses         []AnatomyVote `json:"base_classes,omitempty"`
	Mixins              []AnatomyVote `json:"mixins,omitempty"`
	Methods             []AnatomyVote `json:"methods,omitempty"`
	ConstructorDeps     []AnatomyVote `json:"constructor_deps,omitempty"`
	Imports             []AnatomyVote `json:"imports,omitempty"`
	Capabilities        []AnatomyVote `json:"capabilities,omitempty"`
	HasFirebaseTracking AnatomyVote   `json:"has_firebase_tracking,omitempty"`
}

// AnatomyVote stores how often an anatomy item appears.
type AnatomyVote struct {
	Name       string  `json:"name,omitempty"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
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
