package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplainCommand_AI(t *testing.T) {
	setupScannedDir(t)
	explainAI = true
	explainFeature = ""
	explainRole = ""
	t.Cleanup(func() { explainAI = false })

	buf := &bytes.Buffer{}
	explainCmd.SetOut(buf)
	require.NoError(t, runExplain(explainCmd, nil))

	assert.Contains(t, buf.String(), "## Summary")
	assert.Contains(t, buf.String(), "## Detected Conventions")
	assert.Contains(t, buf.String(), "## Suggested Next Commands")
}

func TestMapCommand_GeneratesFilesAndAI(t *testing.T) {
	setupScannedDir(t)
	mapAI = true
	mapFeature = ""
	mapRender = false
	mapOpen = false
	t.Cleanup(func() { mapAI = false })

	buf := &bytes.Buffer{}
	mapCmd.SetOut(buf)
	require.NoError(t, runMap(mapCmd, nil))

	assert.FileExists(t, filepath.Join(".repox", "maps", "project.md"))
	assert.FileExists(t, filepath.Join(".repox", "maps", "conventions.md"))
	assert.Contains(t, buf.String(), "## Summary")
	assert.Contains(t, buf.String(), "Generated Files")
}

func TestNewFeatureCommand_PreviewAlias(t *testing.T) {
	setupScannedDir(t)
	generateForce = false
	generateDryRun = false
	generatePreview = true
	generateTemplate = ""
	generatePattern = ""
	generateRoles = ""
	generateLike = ""
	generateAI = false
	t.Cleanup(func() { generatePreview = false })

	buf := &bytes.Buffer{}
	newFeatureCmd.SetOut(buf)
	require.NoError(t, runGenerateFeature(newFeatureCmd, []string{"watchlist"}))

	assert.Contains(t, buf.String(), "Dry run")
	_, err := os.Stat(filepath.Join("lib", "features", "watchlist"))
	assert.True(t, os.IsNotExist(err))
}

func TestGenerateDryRunAI(t *testing.T) {
	setupScannedDir(t)
	generateForce = false
	generateDryRun = true
	generatePreview = false
	generateTemplate = ""
	generatePattern = ""
	generateRoles = ""
	generateLike = ""
	generateAI = true
	t.Cleanup(func() {
		generateDryRun = false
		generateAI = false
	})

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	require.NoError(t, runGenerateFeature(generateFeatureCmd, []string{"watchlist"}))

	assert.Contains(t, buf.String(), "## Summary")
	assert.Contains(t, buf.String(), "Files to create")
}
