package retriever

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func defaultConv(featureRoot string) *models.Convention {
	return &models.Convention{
		FeatureRoot: featureRoot,
		Naming: models.NamingConvention{
			ScreenSuffix:     "Screen",
			BlocSuffix:       "Bloc",
			EventSuffix:      "Event",
			StateSuffix:      "State",
			RepositorySuffix: "Repository",
			UsecaseSuffix:    "UseCase",
		},
	}
}

func buildFlutterFeature(t *testing.T, dir, featureName string) {
	t.Helper()
	base := filepath.Join(dir, featureName)
	dartImport := "import 'package:flutter_bloc/flutter_bloc.dart';\n"
	for _, f := range []string{
		featureName + "_screen.dart",
		featureName + "_bloc.dart",
		featureName + "_event.dart",
		featureName + "_state.dart",
		featureName + "_repository.dart",
	} {
		writeFile(t, filepath.Join(base, f), dartImport)
	}
}

// ── IndexFeatures ─────────────────────────────────────────────────────────────

func TestIndexFeatures_BasicFlutter(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	buildFlutterFeature(t, filepath.Join(dir, featureRoot), "home")
	buildFlutterFeature(t, filepath.Join(dir, featureRoot), "profile")

	examples, err := IndexFeatures(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	assert.Len(t, examples, 2)
	// sorted by name
	assert.Equal(t, "home", examples[0].Name)
	assert.Equal(t, "profile", examples[1].Name)
}

func TestIndexFeatures_Empty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "lib/features"), 0o755))

	examples, err := IndexFeatures(dir, defaultConv("lib/features"))
	require.NoError(t, err)
	assert.Empty(t, examples)
}

func TestIndexFeatures_MissingFeatureRoot(t *testing.T) {
	dir := t.TempDir()
	examples, err := IndexFeatures(dir, defaultConv("lib/features"))
	require.NoError(t, err) // not an error — feature root just doesn't exist yet
	assert.Empty(t, examples)
}

func TestIndexFeatures_DetectsMetadata(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	buildFlutterFeature(t, filepath.Join(dir, featureRoot), "home")

	examples, err := IndexFeatures(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)

	meta := examples[0].Metadata
	assert.True(t, meta.HasBloc)
	assert.True(t, meta.HasScreen)
	assert.True(t, meta.HasRepository)
	assert.False(t, meta.HasUseCase) // no usecase file was created
}

func TestIndexFeatures_DetectsPatterns(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	content := "import 'package:flutter_bloc/flutter_bloc.dart';\n" +
		"import 'package:equatable/equatable.dart';\n"
	writeFile(t, filepath.Join(dir, featureRoot, "home", "home_bloc.dart"), content)

	examples, err := IndexFeatures(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)
	assert.Contains(t, examples[0].Patterns, "uses flutter_bloc")
	assert.Contains(t, examples[0].Patterns, "uses equatable")
}

func TestIndexFeatures_FilesRoles(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	buildFlutterFeature(t, filepath.Join(dir, featureRoot), "home")

	examples, err := IndexFeatures(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)

	files := examples[0].Files
	assert.Contains(t, files["bloc"], "home_bloc.dart")
	assert.Contains(t, files["screen"], "home_screen.dart")
	assert.Contains(t, files["repository"], "home_repository.dart")
}

func TestIndexFeatures_DetectsNestedFlowFeatures(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	writeFile(t, filepath.Join(dir, featureRoot, "investment/fund_list/presentation/bloc/fund_list_bloc.dart"), "import 'package:flutter_bloc/flutter_bloc.dart';\n")
	writeFile(t, filepath.Join(dir, featureRoot, "investment/fund_list/presentation/screen/fund_list_screen.dart"), "")
	writeFile(t, filepath.Join(dir, featureRoot, "investment/fund_detail/fund_detail_screen.dart"), "")

	examples, err := IndexFeatures(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 2)

	byPath := map[string]models.Example{}
	for _, ex := range examples {
		byPath[ex.Path] = ex
	}

	fundList := byPath["lib/features/investment/fund_list"]
	assert.Equal(t, "fund_list", fundList.Name)
	assert.True(t, fundList.Metadata.HasBloc)
	assert.True(t, fundList.Metadata.HasScreen)
	assert.Contains(t, fundList.Files["bloc"], "fund_list_bloc.dart")
	assert.NotContains(t, byPath, "lib/features/investment")
}

// ── ScoreSimilarity ───────────────────────────────────────────────────────────

func TestScoreSimilarity_NameOverlap(t *testing.T) {
	profile := models.Example{
		Name:     "profile",
		Metadata: models.FeatureMetadata{HasBloc: true, HasScreen: true},
		Patterns: []string{"uses flutter_bloc"},
	}
	settings := models.Example{
		Name:     "settings",
		Metadata: models.FeatureMetadata{HasBloc: true, HasScreen: true},
		Patterns: []string{"uses flutter_bloc"},
	}
	scoreProfile := ScoreSimilarity("home_profile", profile)
	scoreSettings := ScoreSimilarity("home_profile", settings)
	assert.Greater(t, scoreProfile, scoreSettings, "profile should score higher than settings for target 'home_profile'")
}

func TestScoreSimilarity_ZeroForEmpty(t *testing.T) {
	empty := models.Example{Name: "empty"}
	score := ScoreSimilarity("home", empty)
	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

// ── FindSimilar ────────────────────────────────────────────────────────────────

func TestFindSimilar_TopN(t *testing.T) {
	examples := make([]models.Example, 5)
	for i := range examples {
		examples[i] = models.Example{Name: "feature", Patterns: []string{"uses flutter_bloc"}}
	}
	result := FindSimilar("feature", examples, 3)
	assert.Len(t, result, 3)
}

func TestFindSimilar_AllFeaturesIdentical(t *testing.T) {
	// Edge case: all examples have identical scores — FindSimilar must still return stable topN
	identical := models.Example{
		Name:     "feature",
		Metadata: models.FeatureMetadata{HasBloc: true, HasScreen: true},
		Patterns: []string{"uses flutter_bloc"},
	}
	examples := []models.Example{identical, identical, identical, identical}
	result := FindSimilar("anything", examples, 3)
	assert.Len(t, result, 3)
	// All scores are the same — just verify no panic and correct count
	for _, ex := range result {
		assert.Equal(t, "feature", ex.Name)
	}
}

func TestFindSimilar_FewerThanN(t *testing.T) {
	examples := []models.Example{
		{Name: "home"},
		{Name: "profile"},
	}
	result := FindSimilar("settings", examples, 5)
	assert.Len(t, result, 2)
}

func TestFindSimilar_Empty(t *testing.T) {
	result := FindSimilar("home", nil, 3)
	assert.Empty(t, result)
}

func TestFindSimilar_SortedDescending(t *testing.T) {
	examples := []models.Example{
		{Name: "payments", Metadata: models.FeatureMetadata{HasBloc: true, HasScreen: true, HasRepository: true}},
		{Name: "home", Metadata: models.FeatureMetadata{HasBloc: true, HasScreen: true}},
		{Name: "auth", Metadata: models.FeatureMetadata{HasBloc: true}},
	}
	result := FindSimilar("payments", examples, 3)
	require.Len(t, result, 3)
	// payments should be first since name matches
	assert.Equal(t, "payments", result[0].Name)
}

// ── FeatureIndexer (Retriever interface) ─────────────────────────────────────

func TestFeatureIndexer_ImplementsRetriever(t *testing.T) {
	var _ Retriever = (*FeatureIndexer)(nil)
}

func TestFeatureIndexer_Index(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "lib/features"
	buildFlutterFeature(t, filepath.Join(dir, featureRoot), "home")

	idx := &FeatureIndexer{}
	examples, err := idx.Index(dir, defaultConv(featureRoot))
	require.NoError(t, err)
	assert.Len(t, examples, 1)
}

func TestFeatureIndexer_FindSimilar(t *testing.T) {
	examples := []models.Example{{Name: "home"}, {Name: "profile"}}
	idx := &FeatureIndexer{}
	result := idx.FindSimilar("home", examples, 2)
	assert.Len(t, result, 2)
}

// ── Go indexing ───────────────────────────────────────────────────────────────

func goConv(featureRoot string) *models.Convention {
	return &models.Convention{
		ProjectType: "go",
		FeatureRoot: featureRoot,
		Naming: models.NamingConvention{
			HandlerSuffix:    "Handler",
			ServiceSuffix:    "Service",
			RepositorySuffix: "Repository",
		},
	}
}

func buildGoFeature(t *testing.T, dir, featureName string) {
	t.Helper()
	base := filepath.Join(dir, featureName)
	ginImport := "import (\n\t\"github.com/gin-gonic/gin\"\n)\n"
	for _, f := range []string{
		featureName + "_handler.go",
		featureName + "_service.go",
		featureName + "_repository.go",
		featureName + "_repository_impl.go",
	} {
		writeFile(t, filepath.Join(base, f), "package "+featureName+"\n\n"+ginImport)
	}
	writeFile(t, filepath.Join(base, featureName+"_handler_test.go"), "package "+featureName+"_test\n")
}

func TestIndexFeatures_BasicGo(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	buildGoFeature(t, filepath.Join(dir, featureRoot), "payment")
	buildGoFeature(t, filepath.Join(dir, featureRoot), "order")

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	assert.Len(t, examples, 2)
	assert.Equal(t, "order", examples[0].Name)
	assert.Equal(t, "payment", examples[1].Name)
}

func TestIndexFeatures_GoDetectsMetadata(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	buildGoFeature(t, filepath.Join(dir, featureRoot), "payment")

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)

	meta := examples[0].Metadata
	assert.True(t, meta.HasHandler)
	assert.True(t, meta.HasService)
	assert.True(t, meta.HasRepository)
	assert.True(t, meta.HasTest)
	assert.False(t, meta.HasBloc)
	assert.False(t, meta.HasScreen)
}

func TestIndexFeatures_GoDetectsPatterns(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	content := "package payment\n\nimport (\n\t\"github.com/gin-gonic/gin\"\n\t\"gorm.io/gorm\"\n)\n"
	writeFile(t, filepath.Join(dir, featureRoot, "payment", "payment_handler.go"), content)

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)
	assert.Contains(t, examples[0].Patterns, "uses gin")
	assert.Contains(t, examples[0].Patterns, "uses gorm")
}

func TestIndexFeatures_GoFilesRoles(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	buildGoFeature(t, filepath.Join(dir, featureRoot), "payment")

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)

	files := examples[0].Files
	assert.Contains(t, files["handler"], "payment_handler.go")
	assert.Contains(t, files["service"], "payment_service.go")
	assert.Contains(t, files["repository"], "payment_repository.go")
}

func TestIndexFeatures_GoIgnoresTestFiles(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	writeFile(t, filepath.Join(dir, featureRoot, "payment", "payment_handler.go"), "package payment\n")
	writeFile(t, filepath.Join(dir, featureRoot, "payment", "payment_handler_test.go"), "package payment_test\n")

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)

	// test file should not appear as a role, but HasTest should be true
	assert.NotContains(t, examples[0].Files, "handler_test")
	assert.True(t, examples[0].Metadata.HasTest)
}

func TestIndexFeatures_GoCleanArchStructure(t *testing.T) {
	dir := t.TempDir()
	featureRoot := "internal"
	writeFile(t, filepath.Join(dir, featureRoot, "payment", "delivery", "payment_handler.go"), "package delivery\n")
	writeFile(t, filepath.Join(dir, featureRoot, "payment", "usecase", "payment_service.go"), "package usecase\n")

	examples, err := IndexFeatures(dir, goConv(featureRoot))
	require.NoError(t, err)
	require.Len(t, examples, 1)
	assert.Equal(t, "clean_architecture", examples[0].Metadata.Structure)
}

// ── helpers unit tests ────────────────────────────────────────────────────────

func TestHasSuffix(t *testing.T) {
	assert.True(t, hasSuffix("home_screen.dart", "Screen"))
	assert.True(t, hasSuffix("home_bloc.dart", "Bloc"))
	assert.True(t, hasSuffix("home_use_case.dart", "UseCase"))
	assert.False(t, hasSuffix("home_screen.dart", "Page"))
	assert.False(t, hasSuffix("home_screen.dart", ""))
}

func TestToSnakeSuffix(t *testing.T) {
	assert.Equal(t, "screen", toSnakeSuffix("Screen"))
	assert.Equal(t, "bloc", toSnakeSuffix("Bloc"))
	assert.Equal(t, "use_case", toSnakeSuffix("UseCase"))
	assert.Equal(t, "repository", toSnakeSuffix("Repository"))
}
