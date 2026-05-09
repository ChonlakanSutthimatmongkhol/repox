// Package ai provides an abstraction layer for AI-assisted code generation.
package ai

import "github.com/ChonlakanSutthimatmongkhol/repox/internal/models"

// GenerateRequest holds all context needed for AI generation.
type GenerateRequest struct {
	FeatureName    string
	Conventions    *models.Convention
	Examples       []models.Example // top similar features
	Lessons        []models.Lesson  // approved lessons from previous diffs
	TargetFiles    []string         // expected output file paths
	TargetTemplate string           // template name used to filter lessons by scope
	RootDir        string           // needed to read example file contents
}

// GenerateResponse holds AI-generated file contents.
type GenerateResponse struct {
	Files []GeneratedFileContent `json:"files"`
}

// GeneratedFileContent is a single file in an AI response.
type GeneratedFileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Client is the AI provider interface for scaffold generation.
type Client interface {
	Generate(req GenerateRequest) (*GenerateResponse, error)
}

// Caller is a lower-level interface for sending raw prompts and getting text back.
// Used by the learner package for lesson extraction.
type Caller interface {
	Call(systemPrompt, userPrompt string) (string, error)
}
