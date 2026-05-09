package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

func setupInitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	initForce = false
	require.NoError(t, runInit(initCmd, nil))
	return dir
}

func TestGenerateFeature_CreatesFiles(t *testing.T) {
	dir := setupInitDir(t)

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)

	err := runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "created")
	assert.Contains(t, out, "10 created")

	// verify at least the bloc file was written
	matches, err := filepath.Glob(filepath.Join(dir, "lib/features/watchlist/*.dart"))
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "expected dart files under lib/features/watchlist/")
}

func TestGenerateFeature_RefusesOverwrite(t *testing.T) {
	setupInitDir(t)

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""

	generateFeatureCmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runGenerateFeature(generateFeatureCmd, []string{"watchlist"}))

	// Second run without --force
	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "skipped")
	assert.Contains(t, out, "0 created, 10 skipped")
}

func TestGenerateFeature_ForceOverwrites(t *testing.T) {
	setupInitDir(t)

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""
	generateFeatureCmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runGenerateFeature(generateFeatureCmd, []string{"watchlist"}))

	generateForce = true
	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "10 created")
}

func TestGenerateFeature_DryRun(t *testing.T) {
	dir := setupInitDir(t)

	generateForce = false
	generateDryRun = true
	generateTemplate = ""
	generatePattern = ""

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Dry run")
	assert.True(t, strings.Contains(out, "watchlist"))

	// No actual files should have been written
	matches, _ := filepath.Glob(filepath.Join(dir, "lib/**/*.dart"))
	assert.Len(t, matches, 0, "dry-run should not write any files")
}

func TestGenerateFeature_RolesDryRun(t *testing.T) {
	setupInitDir(t)

	generateForce = false
	generateDryRun = true
	generateTemplate = ""
	generatePattern = ""
	generateRoles = "bloc,event,state,screen"
	generateLike = ""
	defer func() {
		generateDryRun = false
		generateRoles = ""
	}()

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"investment/new_feature"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "new_feature_bloc.dart")
	assert.Contains(t, out, "new_feature_screen.dart")
	assert.NotContains(t, out, "new_feature_repository.dart")
	assert.NotContains(t, out, "new_feature_bloc_test.dart")
}

func TestGenerateFeature_LikeUsesFeatureShape(t *testing.T) {
	setupInitDir(t)

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	require.NoError(t, err)
	conv.FeatureStructure = "flat"
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{
		{
			Name:      "fund_list",
			Path:      "lib/features/investment/fund_list",
			Parent:    "investment",
			Structure: "clean_architecture",
			Files: map[string]string{
				"bloc":   "lib/features/investment/fund_list/presentation/fund_list_bloc.dart",
				"event":  "lib/features/investment/fund_list/presentation/fund_list_event.dart",
				"screen": "lib/features/investment/fund_list/presentation/fund_list_screen.dart",
				"state":  "lib/features/investment/fund_list/presentation/fund_list_state.dart",
			},
			FileRoutes: map[string]string{
				"bloc":   "presentation",
				"event":  "presentation",
				"screen": "presentation",
				"state":  "presentation",
			},
		},
	}
	require.NoError(t, config.Save(config.RepoxPath("conventions.json"), conv))

	generateForce = false
	generateDryRun = true
	generateTemplate = ""
	generatePattern = ""
	generateRoles = ""
	generateLike = "investment/fund_list"
	defer func() {
		generateDryRun = false
		generateLike = ""
	}()

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err = runGenerateFeature(generateFeatureCmd, []string{"investment/new_feature"})
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "lib/features/investment/new_feature/presentation/new_feature_bloc.dart")
	assert.Contains(t, out, "lib/features/investment/new_feature/presentation/new_feature_screen.dart")
	assert.NotContains(t, out, "new_feature_repository.dart")
	assert.NotContains(t, out, "new_feature_bloc_test.dart")
}

func TestGenerateFeature_UsesRecommendedPatternFromScan(t *testing.T) {
	dir := setupInitDir(t)

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	require.NoError(t, err)
	conv.FeatureStructure = "flat"
	conv.FeaturesAnalysis.RecommendedPattern = "clean_architecture"
	conv.PatternMappings = config.DefaultPatternMappings()
	require.NoError(t, config.Save(config.RepoxPath("conventions.json"), conv))

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err = runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "created")
	assert.FileExists(t, filepath.Join(dir, "lib/features/watchlist/presentation/bloc/watchlist_bloc.dart"))
	assert.FileExists(t, filepath.Join(dir, "lib/features/watchlist/presentation/screen/watchlist_screen.dart"))
}

func TestGenerateFeature_PatternOverride(t *testing.T) {
	dir := setupInitDir(t)

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	require.NoError(t, err)
	conv.FeaturesAnalysis.RecommendedPattern = "clean_architecture"
	conv.PatternMappings = config.DefaultPatternMappings()
	require.NoError(t, config.Save(config.RepoxPath("conventions.json"), conv))

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = "flat"
	defer func() { generatePattern = "" }()

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err = runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "created")
	assert.FileExists(t, filepath.Join(dir, "lib/features/watchlist/watchlist_bloc.dart"))
	assert.NoFileExists(t, filepath.Join(dir, "lib/features/watchlist/presentation/bloc/watchlist_bloc.dart"))
}

func TestGenerateFeature_NoRepoxDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""

	err := runGenerateFeature(generateFeatureCmd, []string{"watchlist"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repox init")
}
