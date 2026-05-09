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
	featuresAnalysis, err := AnalyzeFeatureRoot(rootDir, featureRoot)
	if err != nil {
		return nil, fmt.Errorf("flutter_scanner: analyze features: %w", err)
	}
	conv.FeaturesAnalysis = featuresAnalysis
	if featuresAnalysis.RecommendedPattern != "" {
		conv.FeatureStructure = featuresAnalysis.RecommendedPattern
	}

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
	conv.PatternMappings = defaultPatternMappings()

	return conv, nil
}

func defaultPatternMappings() models.PatternMappings {
	return models.PatternMappings{
		"flat": {
			FileRoutes: map[string]string{
				"bloc":            "",
				"event":           "",
				"state":           "",
				"screen":          "",
				"repository":      "",
				"repository_impl": "",
				"request":         "",
				"response":        "",
				"usecase":         "",
			},
		},
		"grouped": {
			FileRoutes: map[string]string{
				"bloc":            "bloc",
				"event":           "bloc",
				"state":           "bloc",
				"screen":          "screen",
				"repository":      "repository",
				"repository_impl": "repository",
				"request":         "models",
				"response":        "models",
				"usecase":         "usecase",
			},
		},
		"clean_architecture": {
			FileRoutes: map[string]string{
				"bloc":            "presentation/bloc",
				"event":           "presentation/bloc",
				"state":           "presentation/bloc",
				"screen":          "presentation/screen",
				"repository":      "domain/repositories",
				"repository_impl": "data/repositories",
				"request":         "data/models",
				"response":        "data/models",
				"usecase":         "domain/usecases",
			},
		},
	}
}
