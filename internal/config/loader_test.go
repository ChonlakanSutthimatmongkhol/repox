package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultConfig()
	require.NoError(t, Save(path, cfg))

	loaded, err := Load[models.Config](path)
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{bad json"), 0o644))

	_, err := Load[models.Config](path)
	assert.Error(t, err)
}

func TestLoad_Missing(t *testing.T) {
	_, err := Load[models.Config]("/nonexistent/path/config.json")
	assert.Error(t, err)
}

func TestDefaultConventions(t *testing.T) {
	conv := DefaultConventions()
	assert.Equal(t, "PascalCase", conv.Naming.ClassCase)
	assert.Equal(t, "snake_case", conv.Naming.FileCase)
	assert.Equal(t, "Bloc", conv.Naming.BlocSuffix)
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.json")
	require.NoError(t, Save(path, DefaultConfig()))

	loaded, err := Load[models.Config](path)
	require.NoError(t, err)
	assert.Equal(t, "0.1.0", loaded.Version)
}
