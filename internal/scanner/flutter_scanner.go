package scanner

import (
	"fmt"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// FlutterScanner implements Scanner for Flutter projects.
type FlutterScanner struct{}

// Scan detects all conventions for a Flutter project at rootDir.
func (s *FlutterScanner) Scan(rootDir string) (*models.Convention, error) {
	conv := &models.Convention{
		ProjectType: "flutter",
	}

	featureRoot, err := DetectFeatureRoot(rootDir, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect feature root: %w", err)
	}
	if featureRoot == "" {
		featureRoot = "lib/features"
	}
	conv.FeatureRoot = featureRoot

	structure, err := DetectFeatureStructure(rootDir + "/" + featureRoot)
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect structure: %w", err)
	}
	conv.FeatureStructure = structure

	testRoot, err := DetectTestRoot(rootDir, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect test root: %w", err)
	}
	if testRoot == "" {
		testRoot = "test"
	}
	conv.TestRoot = testRoot

	naming, err := DetectNamingConventions(rootDir+"/"+featureRoot, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect naming: %w", err)
	}
	conv.Naming = naming

	imports, err := DetectCommonImports(rootDir+"/"+featureRoot, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect imports: %w", err)
	}
	conv.CommonImports = imports

	stateManagement, err := DetectStateManagement(rootDir, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect state management: %w", err)
	}
	conv.StateManagement = stateManagement

	routing, err := DetectRouting(rootDir, "flutter")
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: detect routing: %w", err)
	}
	conv.Routing = routing

	return conv, nil
}
