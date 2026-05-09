// Package scanner detects project conventions by analyzing the repository.
package scanner

import "github.com/ChonlakanSutthimatmongkhol/repox/internal/models"

// Scanner analyzes a repository and returns detected conventions.
type Scanner interface {
	Scan(rootDir string) (*models.Convention, error)
}

// ScanResult holds partial results accumulated by individual sub-scanners.
type ScanResult struct {
	ProjectType      string
	StateManagement  string
	FeatureStructure string
	FeatureRoot      string
	TestRoot         string
	Naming           models.NamingConvention
	Routing          models.RoutingConfig
	CommonImports    []string
}

// excludedDirs lists directories that should never be scanned.
var excludedDirs = map[string]bool{
	".repox":      true,
	"build":       true,
	".dart_tool":  true,
	"node_modules": true,
	".git":        true,
}
