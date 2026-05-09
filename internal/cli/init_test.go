package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)

	initForce = false
	err := runInit(initCmd, nil)
	require.NoError(t, err)

	expected := []string{
		".repox/config.json",
		".repox/conventions.json",
		".repox/examples.json",
		".repox/lessons.json",
		".repox/generations.json",
	}
	for _, f := range expected {
		_, err := os.Stat(filepath.Join(dir, f))
		assert.NoError(t, err, "expected file %s to exist", f)
	}
}

func TestInitCommand_RefusesIfExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	// Second call without --force should fail
	initForce = false
	err := runInit(initCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
}

func TestInitCommand_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	initForce = true
	err := runInit(initCmd, nil)
	assert.NoError(t, err)
}
