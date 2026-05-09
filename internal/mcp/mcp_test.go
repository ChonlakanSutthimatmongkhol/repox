package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func buildRepoxDir(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.Chdir(dir))

	repoxDir := filepath.Join(dir, ".repox")
	require.NoError(t, os.MkdirAll(repoxDir, 0o755))

	cfg := config.DefaultConfig()
	require.NoError(t, config.Save(config.RepoxPath("config.json"), cfg))
	require.NoError(t, config.Save(config.RepoxPath("conventions.json"), config.DefaultConventions()))
	require.NoError(t, config.Save(config.RepoxPath("examples.json"), []models.Example{}))
	require.NoError(t, config.Save(config.RepoxPath("lessons.json"), []models.Lesson{}))
	require.NoError(t, config.Save(config.RepoxPath("generations.json"), []models.Generation{}))
}

func makeReq(args map[string]any) mcplib.CallToolRequest {
	return mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Arguments: args,
		},
	}
}

// ── argsMap ───────────────────────────────────────────────────────────────────

func TestArgsMap_NilInput(t *testing.T) {
	m := argsMap(nil)
	assert.NotNil(t, m)
	assert.Empty(t, m)
}

func TestArgsMap_ValidMap(t *testing.T) {
	m := argsMap(map[string]any{"key": "val"})
	assert.Equal(t, "val", m["key"])
}

func TestArgsMap_WrongType(t *testing.T) {
	m := argsMap("not a map")
	assert.NotNil(t, m)
	assert.Empty(t, m)
}

// ── optionalString / optionalBool / optionalInt ───────────────────────────────

func TestOptionalString(t *testing.T) {
	assert.Equal(t, "flutter", optionalString(map[string]any{"k": "flutter"}, "k"))
	assert.Equal(t, "", optionalString(map[string]any{}, "k"))
}

func TestOptionalBool(t *testing.T) {
	assert.True(t, optionalBool(map[string]any{"k": true}, "k"))
	assert.False(t, optionalBool(map[string]any{}, "k"))
}

func TestOptionalInt(t *testing.T) {
	assert.Equal(t, 5, optionalInt(map[string]any{"k": float64(5)}, "k", "3"))
	assert.Equal(t, 3, optionalInt(map[string]any{}, "k", "3")) // missing → default 3
}

// ── handleExplainConvention ───────────────────────────────────────────────────

func TestHandleExplainConvention_NoRepoxDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	result, err := handleExplainConvention(nil, makeReq(nil))
	require.NoError(t, err) // handler never returns Go errors, wraps in CallToolResult
	assert.True(t, result.IsError)
}

func TestHandleExplainConvention_WithConventions(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)

	result, err := handleExplainConvention(nil, makeReq(nil))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcplib.TextContent).Text
	assert.Contains(t, text, "flutter")
	assert.Contains(t, text, "Feature root")
}

// ── handleFindSimilar ─────────────────────────────────────────────────────────

func TestHandleFindSimilar_NoFeatureName(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)

	result, err := handleFindSimilar(nil, makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleFindSimilar_NoExamples(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)
	// No features in repo — expect "No similar features found."
	result, err := handleFindSimilar(nil, makeReq(map[string]any{"feature_name": "payments"}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcplib.TextContent).Text
	assert.Contains(t, text, "No similar features found")
}

// ── handleGenerate ────────────────────────────────────────────────────────────

func TestHandleGenerate_DryRun(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)

	result, err := handleGenerate(nil, makeReq(map[string]any{
		"feature_name": "watchlist",
		"dry_run":      true,
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcplib.TextContent).Text
	assert.Contains(t, text, "Dry run")
	assert.Contains(t, text, "watchlist")
}

func TestHandleGenerate_TemplateMode(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)

	result, err := handleGenerate(nil, makeReq(map[string]any{
		"feature_name": "payments",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcplib.TextContent).Text
	assert.Contains(t, text, "created")

	// Files should exist
	matches, _ := filepath.Glob(filepath.Join(dir, "lib/features/payments/*.dart"))
	assert.Greater(t, len(matches), 0)
}

// ── handleScan ────────────────────────────────────────────────────────────────

func TestHandleScan_NoRepoxDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	result, err := handleScan(nil, makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHandleScan_UnsupportedType(t *testing.T) {
	dir := t.TempDir()
	buildRepoxDir(t, dir)

	result, err := handleScan(nil, makeReq(map[string]any{"project_override": "rust"}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

// ── handleLearn ───────────────────────────────────────────────────────────────

func TestHandleLearn_ReturnsCliHint(t *testing.T) {
	result, err := handleLearn(nil, makeReq(nil))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	text := result.Content[0].(mcplib.TextContent).Text
	assert.Contains(t, text, "repox learn")
}
