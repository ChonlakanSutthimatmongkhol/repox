package models

// Config is the main repox project configuration stored in .repox/config.json.
type Config struct {
	Version         string   `json:"version"`
	ProjectType     string   `json:"project_type"`
	FeatureRoot     string   `json:"feature_root"`
	TestRoot        string   `json:"test_root"`
	DefaultTemplate string   `json:"default_template"`
	AI              AIConfig `json:"ai"`
}

// AIConfig holds AI provider settings.
type AIConfig struct {
	Provider        string `json:"provider"`
	GenerationModel string `json:"generation_model"`
	LearningModel   string `json:"learning_model"`
}
