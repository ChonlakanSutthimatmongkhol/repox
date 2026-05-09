package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ── Mock client ──────────────────────────────────────────────────────────────

type mockClient struct {
	resp *GenerateResponse
	err  error
}

func (m *mockClient) Generate(_ GenerateRequest) (*GenerateResponse, error) {
	return m.resp, m.err
}

var _ Client = (*mockClient)(nil)

// ── Response Parser ──────────────────────────────────────────────────────────

func TestParseResponse_ValidJSON(t *testing.T) {
	raw := `{"files":[{"path":"lib/features/home/home_bloc.dart","content":"class HomeBloc {}"}]}`
	resp, err := ParseResponse(raw)
	require.NoError(t, err)
	require.Len(t, resp.Files, 1)
	assert.Equal(t, "lib/features/home/home_bloc.dart", resp.Files[0].Path)
	assert.Equal(t, "class HomeBloc {}", resp.Files[0].Content)
}

func TestParseResponse_StripsFences(t *testing.T) {
	raw := "```json\n{\"files\":[{\"path\":\"a.dart\",\"content\":\"x\"}]}\n```"
	resp, err := ParseResponse(raw)
	require.NoError(t, err)
	assert.Len(t, resp.Files, 1)
}

func TestParseResponse_StripsFencesNoLang(t *testing.T) {
	raw := "```\n{\"files\":[{\"path\":\"a.dart\",\"content\":\"x\"}]}\n```"
	resp, err := ParseResponse(raw)
	require.NoError(t, err)
	assert.Len(t, resp.Files, 1)
}

func TestParseResponse_InvalidJSON(t *testing.T) {
	_, err := ParseResponse("not json")
	assert.Error(t, err)
}

func TestParseResponse_EmptyPath(t *testing.T) {
	raw := `{"files":[{"path":"","content":"x"}]}`
	_, err := ParseResponse(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty path")
}

func TestParseResponse_EmptyContent(t *testing.T) {
	raw := `{"files":[{"path":"a.dart","content":""}]}`
	_, err := ParseResponse(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty content")
}

func TestParseResponse_NoFiles(t *testing.T) {
	raw := `{"files":[]}`
	_, err := ParseResponse(raw)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no files")
}

// ── Prompt Builder ───────────────────────────────────────────────────────────

func TestBuildSystemPrompt_ContainsRules(t *testing.T) {
	prompt := BuildSystemPrompt()
	assert.Contains(t, prompt, "Repox")
	assert.Contains(t, prompt, "Return ONLY valid JSON")
	assert.Contains(t, prompt, "naming conventions exactly")
}

func TestBuildUserPrompt_ContainsFeatureName(t *testing.T) {
	req := GenerateRequest{
		FeatureName: "watchlist",
		Conventions: &models.Convention{ProjectType: "flutter"},
		TargetFiles: []string{"lib/features/watchlist/watchlist_bloc.dart"},
	}
	prompt := BuildUserPrompt(req)
	assert.Contains(t, prompt, "watchlist")
	assert.Contains(t, prompt, "lib/features/watchlist/watchlist_bloc.dart")
	assert.Contains(t, prompt, "Return ONLY valid JSON")
}

func TestBuildUserPrompt_IncludesLessons(t *testing.T) {
	req := GenerateRequest{
		FeatureName:    "home",
		Conventions:    &models.Convention{},
		TargetTemplate: "flutter_bloc_feature",
		Lessons: []models.Lesson{
			{
				ID:         "lesson_1",
				Scope:      "flutter_bloc_feature",
				Lesson:     "always extend BaseBloc",
				Confidence: 0.9,
				Approved:   true,
				Source:     models.LessonSource{DetectedFrom: "manual"},
			},
		},
	}
	prompt := BuildUserPrompt(req)
	assert.Contains(t, prompt, "Lessons learned")
	assert.Contains(t, prompt, "always extend BaseBloc")
}

func TestBuildUserPrompt_NoExamples(t *testing.T) {
	req := GenerateRequest{
		FeatureName: "home",
		Conventions: &models.Convention{},
	}
	prompt := BuildUserPrompt(req)
	assert.Contains(t, prompt, "no examples available")
}

// ── StripFences ───────────────────────────────────────────────────────────────

func TestStripFences_NoFence(t *testing.T) {
	assert.Equal(t, `{"files":[]}`, stripFences(`{"files":[]}`))
}

func TestStripFences_WithLang(t *testing.T) {
	input := "```json\n{\"files\":[]}\n```"
	assert.Equal(t, `{"files":[]}`, stripFences(input))
}

func TestStripFences_WithoutLang(t *testing.T) {
	input := "```\n{\"files\":[]}\n```"
	assert.Equal(t, `{"files":[]}`, stripFences(input))
}

// ── Mock client integration ──────────────────────────────────────────────────

func TestMockClient_ReturnsResponse(t *testing.T) {
	mc := &mockClient{
		resp: &GenerateResponse{
			Files: []GeneratedFileContent{
				{Path: "lib/features/home/home_bloc.dart", Content: "class HomeBloc {}"},
			},
		},
	}
	req := GenerateRequest{FeatureName: "home", Conventions: &models.Convention{}}
	resp, err := mc.Generate(req)
	require.NoError(t, err)
	assert.Len(t, resp.Files, 1)
}

func TestMockClient_ImplementsInterface(t *testing.T) {
	var _ Client = (*mockClient)(nil)
}

// ── BuildUserPrompt token budget ─────────────────────────────────────────────

func TestBuildUserPrompt_LargeExamplesDoNotPanic(t *testing.T) {
	// Generate a very large "example" to trigger token budget truncation
	bigContent := strings.Repeat("a", 200_000)
	req := GenerateRequest{
		FeatureName: "home",
		Conventions: &models.Convention{},
		Examples: []models.Example{
			{
				Name:  "home",
				Files: map[string]string{"bloc": "nonexistent_file.dart"},
				Metadata: models.FeatureMetadata{
					Imports: []string{bigContent},
				},
			},
		},
	}
	// Should not panic; prompt may truncate examples
	prompt := BuildUserPrompt(req)
	assert.NotEmpty(t, prompt)
}
