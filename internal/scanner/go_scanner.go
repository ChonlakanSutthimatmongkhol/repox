package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// GoScanner implements Scanner for Go projects.
type GoScanner struct{}

// Scan detects all conventions for a Go project at rootDir.
func (s *GoScanner) Scan(rootDir string) (*models.Convention, error) {
	conv := &models.Convention{
		ProjectType:     "go",
		StateManagement: "none",
	}

	modulePath, err := DetectGoModule(rootDir)
	if err == nil {
		conv.ModulePath = modulePath
	}

	featureRoot, err := DetectFeatureRoot(rootDir, "go")
	if err != nil {
		return nil, fmt.Errorf("go_scanner: detect feature root: %w", err)
	}
	if featureRoot == "" {
		featureRoot = "internal"
	}
	conv.FeatureRoot = featureRoot

	featuresAnalysis, err := AnalyzeFeatureRoot(rootDir, featureRoot)
	if err != nil {
		return nil, fmt.Errorf("go_scanner: analyze features: %w", err)
	}
	conv.FeaturesAnalysis = featuresAnalysis

	structure := featuresAnalysis.RecommendedPattern
	if structure == "" {
		structure, _ = DetectFeatureStructure(filepath.Join(rootDir, featureRoot))
	}
	if structure == "" || structure == "grouped" {
		structure = "flat"
	}
	conv.FeatureStructure = structure

	// Go tests live alongside source files; use same root.
	conv.TestRoot = featureRoot

	naming, err := DetectNamingConventions(filepath.Join(rootDir, featureRoot), "go")
	if err != nil {
		return nil, fmt.Errorf("go_scanner: detect naming: %w", err)
	}
	conv.Naming = naming

	imports, err := DetectCommonImports(filepath.Join(rootDir, featureRoot), "go")
	if err != nil {
		return nil, fmt.Errorf("go_scanner: detect imports: %w", err)
	}
	conv.CommonImports = imports

	routing, err := DetectGoRouting(rootDir)
	if err != nil {
		return nil, fmt.Errorf("go_scanner: detect routing: %w", err)
	}
	conv.Routing = routing

	conv.PatternMappings = goPatternMappings()

	return conv, nil
}

// DetectGoModule reads the module path from go.mod.
func DetectGoModule(rootDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("go_scanner: module directive not found in go.mod")
}

// DetectGoRouting returns the HTTP framework in use based on go.mod dependencies.
func DetectGoRouting(rootDir string) (models.RoutingConfig, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.mod"))
	if err != nil {
		return models.RoutingConfig{Type: "net/http"}, nil
	}
	content := string(data)

	var frameworkType string
	switch {
	case strings.Contains(content, "github.com/gin-gonic/gin"):
		frameworkType = "gin"
	case strings.Contains(content, "github.com/go-chi/chi"):
		frameworkType = "chi"
	case strings.Contains(content, "github.com/labstack/echo"):
		frameworkType = "echo"
	case strings.Contains(content, "github.com/gofiber/fiber"):
		frameworkType = "fiber"
	case strings.Contains(content, "github.com/gorilla/mux"):
		frameworkType = "gorilla_mux"
	default:
		frameworkType = "net/http"
	}

	return models.RoutingConfig{Type: frameworkType}, nil
}

func goPatternMappings() models.PatternMappings {
	return models.PatternMappings{
		"flat": {
			FileRoutes: map[string]string{
				"handler":         "",
				"service":         "",
				"repository":      "",
				"repository_impl": "",
				"model":           "",
				"handler_test":    "",
			},
		},
		"clean_architecture": {
			FileRoutes: map[string]string{
				"handler":         "delivery",
				"service":         "usecase",
				"repository":      "repository",
				"repository_impl": "repository",
				"model":           "domain",
				"handler_test":    "delivery",
			},
		},
	}
}
