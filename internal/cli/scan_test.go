package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func buildFlutterProjectForCLI(t *testing.T, dir string) {
	t.Helper()
	pubspec := `
name: myapp
dependencies:
  flutter:
    sdk: flutter
  flutter_bloc: ^8.0.0
  go_router: ^12.0.0
flutter:
  uses-material-design: true
`
	writeTestFile(t, filepath.Join(dir, "pubspec.yaml"), pubspec)

	dartImport := "import 'package:flutter_bloc/flutter_bloc.dart';\n"
	files := []string{
		"lib/features/home/presentation/screen/home_screen.dart",
		"lib/features/home/presentation/bloc/home_bloc.dart",
		"lib/features/home/presentation/bloc/home_event.dart",
		"lib/features/home/presentation/bloc/home_state.dart",
		"lib/features/home/domain/repository/home_repository.dart",
		"lib/features/profile/presentation/screen/profile_screen.dart",
		"lib/features/profile/presentation/bloc/profile_bloc.dart",
		"lib/router/app_router.dart",
		"test/features/home/home_bloc_test.dart",
	}
	for _, f := range files {
		writeTestFile(t, filepath.Join(dir, f), dartImport)
	}
}

func TestScanCommand_FlutterProject(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buildFlutterProjectForCLI(t, dir)

	// repox init
	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	// repox scan
	scanProjectOverride = ""
	scanDeep = true
	buf := &bytes.Buffer{}
	scanCmd.SetOut(buf)
	err := runScan(scanCmd, nil)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "flutter")
	assert.Contains(t, out, "flutter_bloc")
	assert.Contains(t, out, "lib/features")
	assert.Contains(t, out, "Pattern Analysis:")
	assert.Contains(t, out, "Total features: 2")
	assert.Contains(t, out, "clean_architecture: 1 features (50.0%)")
	assert.Contains(t, out, "grouped: 1 features (50.0%)")
	assert.Contains(t, out, "Recommended pattern: grouped")
	assert.Contains(t, out, "Latest pattern:")
	assert.Contains(t, out, "repox generate feature <name>")
	assert.Contains(t, out, "Conventions saved")

	// Verify conventions.json was written correctly
	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	require.NoError(t, err)
	assert.Equal(t, "flutter", conv.ProjectType)
	assert.Equal(t, "flutter_bloc", conv.StateManagement)
	assert.Equal(t, "lib/features", conv.FeatureRoot)
	assert.Equal(t, "grouped", conv.FeatureStructure)
	assert.Equal(t, "test/features", conv.TestRoot)
	assert.Equal(t, "Screen", conv.Naming.ScreenSuffix)
	assert.Equal(t, "Bloc", conv.Naming.BlocSuffix)
	assert.Equal(t, "go_router", conv.Routing.Type)
	assert.Equal(t, "lib/router/app_router.dart", conv.Routing.RouteFile)
	require.Len(t, conv.FeaturesAnalysis.Features, 2)
	assert.Equal(t, 1, conv.FeaturesAnalysis.PatternDistribution["clean_architecture"].Count)
	assert.Equal(t, 1, conv.FeaturesAnalysis.PatternDistribution["grouped"].Count)
	assert.Equal(t, "grouped", conv.FeaturesAnalysis.RecommendedPattern)
	assert.NotEmpty(t, conv.FeaturesAnalysis.LatestPattern)
	assert.Contains(t, conv.PatternMappings, "flat")
	assert.Contains(t, conv.PatternMappings, "clean_architecture")
}

func TestScanCommand_UpdatesConfigJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buildFlutterProjectForCLI(t, dir)
	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	scanProjectOverride = ""
	scanCmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runScan(scanCmd, nil))

	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	require.NoError(t, err)
	assert.Equal(t, "flutter", cfg.ProjectType)
	assert.Equal(t, "lib/features", cfg.FeatureRoot)
}

func TestScanCommand_NoRepoxDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	scanProjectOverride = ""
	err := runScan(scanCmd, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repox init")
}

func TestScanCommand_ProjectOverride(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	// Flutter files present but we override to trigger the unsupported path
	writeTestFile(t, filepath.Join(dir, "pubspec.yaml"), "flutter:\n  sdk: flutter\n")
	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	scanProjectOverride = "unknown_type"
	buf := &bytes.Buffer{}
	scanCmd.SetOut(buf)
	err := runScan(scanCmd, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Warning")
}

// Integration: scan indexes features and saves examples.json
func TestScanIndexesExamples(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buildFlutterProjectForCLI(t, dir)
	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	scanProjectOverride = ""
	buf := &bytes.Buffer{}
	scanCmd.SetOut(buf)
	require.NoError(t, runScan(scanCmd, nil))

	// examples.json must exist and contain at least one entry
	examplesPath := filepath.Join(dir, ".repox", "examples.json")
	data, err := os.ReadFile(examplesPath)
	require.NoError(t, err, "examples.json should be created by repox scan")
	assert.Contains(t, string(data), "home", "examples.json should contain the 'home' feature")

	// output should mention indexed features
	assert.Contains(t, buf.String(), "Indexed")
}

// Integration: scan → index → generate --with-examples shows similar features
func TestScanIndexGenerateWithExamples(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buildFlutterProjectForCLI(t, dir)
	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	scanProjectOverride = ""
	scanCmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runScan(scanCmd, nil))

	// Generate with --with-examples; existing features (home, profile) should appear
	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generateWithExamples = true
	defer func() { generateWithExamples = false }()

	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"payments"})
	require.NoError(t, err)

	out := buf.String()
	// Should print at least one similar feature header (home or profile indexed)
	assert.Contains(t, out, "Similar features found:")
	// Should still generate files
	assert.Contains(t, out, "created")
}

// Integration: scan then generate uses scanned conventions
func TestScanThenGenerate_UsesScannedConventions(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	buildFlutterProjectForCLI(t, dir)

	initForce = false
	require.NoError(t, runInit(initCmd, nil))

	scanProjectOverride = ""
	scanCmd.SetOut(&bytes.Buffer{})
	require.NoError(t, runScan(scanCmd, nil))

	// Generate a new feature — it should use scanned conventions
	generateForce = false
	generateDryRun = false
	generateTemplate = ""
	generatePattern = ""
	buf := &bytes.Buffer{}
	generateFeatureCmd.SetOut(buf)
	err := runGenerateFeature(generateFeatureCmd, []string{"settings"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "created")

	// Verify files land under the scanned feature root
	matches, err := filepath.Glob(filepath.Join(dir, "lib/features/settings/bloc/*.dart"))
	require.NoError(t, err)
	assert.Greater(t, len(matches), 0, "expected dart files under grouped bloc folder")
}
