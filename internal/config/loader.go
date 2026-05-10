// Package config handles loading and saving repox configuration files from .repox/.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

const repoxDir = ".repox"

// Load reads and unmarshals a JSON file at path into v.
func Load[T any](path string) (T, error) {
	var zero T
	data, err := os.ReadFile(path)
	if err != nil {
		return zero, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &zero); err != nil {
		return zero, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return zero, nil
}

// Save marshals v to indented JSON and writes it to path, creating directories as needed.
func Save[T any](path string, v T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// DefaultConfig returns the default repox configuration.
func DefaultConfig() models.Config {
	return models.Config{
		Version:         "0.1.0",
		ProjectType:     "flutter",
		FeatureRoot:     "lib/features",
		TestRoot:        "test/features",
		DefaultTemplate: "flutter_bloc_feature",
	}
}

// DefaultConventions returns the default naming conventions for a Flutter BLoC project.
func DefaultConventions() models.Convention {
	return models.Convention{
		ProjectType:      "flutter",
		StateManagement:  "bloc",
		FeatureStructure: "flat",
		FeatureRoot:      "lib/features",
		TestRoot:         "test/features",
		Naming: models.NamingConvention{
			ClassCase:        "PascalCase",
			FileCase:         "snake_case",
			ScreenSuffix:     "Screen",
			BlocSuffix:       "Bloc",
			EventSuffix:      "Event",
			StateSuffix:      "State",
			RepositorySuffix: "Repository",
			UsecaseSuffix:    "UseCase",
		},
		Routing: models.RoutingConfig{
			Type:      "go_router",
			RouteFile: "lib/app/routes.dart",
		},
		CommonImports: []string{
			"package:flutter/material.dart",
			"package:flutter_bloc/flutter_bloc.dart",
		},
		PatternMappings: DefaultPatternMappings(),
		Roles: map[string]models.RoleConvention{
			"bloc":            {FileSuffix: "bloc", ClassSuffix: "Bloc"},
			"event":           {FileSuffix: "event", ClassSuffix: "Event"},
			"repository":      {FileSuffix: "repository", ClassSuffix: "Repository"},
			"repository_impl": {FileSuffix: "repository_impl", ClassSuffix: "RepositoryImpl"},
			"request":         {FileSuffix: "request", ClassSuffix: "Request"},
			"response":        {FileSuffix: "response", ClassSuffix: "Response"},
			"screen":          {FileSuffix: "screen", ClassSuffix: "Screen"},
			"state":           {FileSuffix: "state", ClassSuffix: "State"},
			"usecase":         {FileSuffix: "usecase", ClassSuffix: "UseCase"},
		},
	}
}

// DefaultPatternMappings returns template routing for supported feature patterns.
func DefaultPatternMappings() models.PatternMappings {
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

// RepoxPath returns the path to a file inside the .repox directory.
func RepoxPath(filename string) string {
	return filepath.Join(repoxDir, filename)
}

// RepoxDirExists reports whether the .repox directory exists in the current working directory.
func RepoxDirExists() bool {
	_, err := os.Stat(repoxDir)
	return err == nil
}
