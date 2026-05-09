package scanner

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// Pubspec is a minimal representation of a Flutter pubspec.yaml.
type Pubspec struct {
	Name         string                 `yaml:"name"`
	Dependencies map[string]interface{} `yaml:"dependencies"`
	DevDeps      map[string]interface{} `yaml:"dev_dependencies"`
	Flutter      map[string]interface{} `yaml:"flutter"`
}

// DetectStateManagement returns the state management framework in use.
// Returns "flutter_bloc", "cubit", "riverpod", "provider", or "unknown".
func DetectStateManagement(rootDir, projectType string) (string, error) {
	if projectType != "flutter" && projectType != "dart" {
		return "unknown", nil
	}

	pubspec, err := loadPubspec(rootDir)
	if err != nil {
		return "unknown", nil
	}

	deps := pubspec.Dependencies
	if deps == nil {
		return "unknown", nil
	}

	switch {
	case hasDep(deps, "flutter_bloc"):
		return "flutter_bloc", nil
	case hasDep(deps, "flutter_riverpod") || hasDep(deps, "riverpod"):
		return "riverpod", nil
	case hasDep(deps, "provider"):
		return "provider", nil
	case hasDep(deps, "bloc"):
		return "flutter_bloc", nil
	}
	return "unknown", nil
}

// DetectRouting returns the routing framework and route file path.
func DetectRouting(rootDir, projectType string) (models.RoutingConfig, error) {
	if projectType != "flutter" && projectType != "dart" {
		return models.RoutingConfig{}, nil
	}

	pubspec, err := loadPubspec(rootDir)
	routingType := "unknown"
	if err == nil && pubspec.Dependencies != nil {
		switch {
		case hasDep(pubspec.Dependencies, "go_router"):
			routingType = "go_router"
		case hasDep(pubspec.Dependencies, "auto_route"):
			routingType = "auto_route"
		}
	}

	routeFileCandidates := []string{
		"lib/router/app_router.dart",
		"lib/routes/routes.dart",
		"lib/app/router.dart",
		"lib/app/routes.dart",
	}
	for _, c := range routeFileCandidates {
		if _, err := os.Stat(filepath.Join(rootDir, c)); err == nil {
			return models.RoutingConfig{Type: routingType, RouteFile: c}, nil
		}
	}

	return models.RoutingConfig{Type: routingType}, nil
}

func loadPubspec(rootDir string) (*Pubspec, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, "pubspec.yaml"))
	if err != nil {
		return nil, err
	}
	var p Pubspec
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func hasDep(deps map[string]interface{}, name string) bool {
	_, ok := deps[name]
	return ok
}
