package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand_NoRepoxDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buf := &bytes.Buffer{}
	doctorCmd.SetOut(buf)
	require.NoError(t, runDoctor(doctorCmd, nil))

	assert.Contains(t, buf.String(), "Repox Doctor")
	assert.Contains(t, buf.String(), ".repox missing")
	assert.Contains(t, buf.String(), "repox setup")
}

func TestDoctorCommand_ReadyState(t *testing.T) {
	setupScannedDir(t)

	buf := &bytes.Buffer{}
	doctorCmd.SetOut(buf)
	require.NoError(t, runDoctor(doctorCmd, nil))

	out := buf.String()
	assert.Contains(t, out, ".repox exists")
	assert.Contains(t, out, "config.json exists")
	assert.Contains(t, out, "Project type")
}

func TestSetupCommand_CreatesSkillAndPrintsNext(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	buildFlutterProjectForCLI(t, dir)

	buf := &bytes.Buffer{}
	setupCmd.SetOut(buf)
	require.NoError(t, runSetup(setupCmd, nil))

	out := buf.String()
	assert.Contains(t, out, "Scanned project conventions")
	assert.Contains(t, out, "Generated .repox/skill/SKILL.md")
	assert.Contains(t, out, "repox map --open")
	assert.FileExists(t, ".repox/skill/SKILL.md")
}

func TestSetupCommand_Idempotent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	buildFlutterProjectForCLI(t, dir)
	require.NoError(t, runSetup(setupCmd, nil))

	buf := &bytes.Buffer{}
	setupCmd.SetOut(buf)
	require.NoError(t, runSetup(setupCmd, nil))

	assert.Contains(t, buf.String(), ".repox already initialized")
	assert.False(t, strings.Contains(buf.String(), "already exists. Use --force"))
}
