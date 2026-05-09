package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
)

func TestGenerate_Watchlist(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()

	files, err := gen.Generate("watchlist", "flutter_bloc_feature", &conv)
	require.NoError(t, err)
	assert.Len(t, files, 10)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	// bloc file should exist and contain Watchlist
	var blocFile string
	for p, c := range byPath {
		if strings.Contains(p, "bloc.dart") && !strings.Contains(p, "test") {
			blocFile = c
			break
		}
	}
	require.NotEmpty(t, blocFile, "bloc file should be generated")
	assert.Contains(t, blocFile, "WatchlistBloc")
	assert.Contains(t, blocFile, "watchlist_event.dart")
	assert.Contains(t, blocFile, "watchlist_state.dart")
}

func TestGenerate_CamelInput(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()

	files, err := gen.Generate("watchList", "flutter_bloc_feature", &conv)
	require.NoError(t, err)

	for _, f := range files {
		// All output paths should use snake_case
		assert.False(t, strings.Contains(f.Path, "watchList"), "path should not contain camelCase: %s", f.Path)
		assert.Contains(t, f.Path, "watch_list")
	}
}

func TestGenerate_CleanArchitectureRoutesAndImports(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "clean_architecture"

	files, err := gen.Generate("watchlist", "flutter_bloc_feature", &conv)
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	screenPath := "lib/features/watchlist/presentation/screen/watchlist_screen.dart"
	blocPath := "lib/features/watchlist/presentation/bloc/watchlist_bloc.dart"
	repoImplPath := "lib/features/watchlist/data/repositories/watchlist_repository_impl.dart"
	usecasePath := "lib/features/watchlist/domain/usecases/watchlist_usecase.dart"

	require.Contains(t, byPath, screenPath)
	require.Contains(t, byPath, blocPath)
	require.Contains(t, byPath, repoImplPath)
	require.Contains(t, byPath, usecasePath)
	assert.Contains(t, byPath[screenPath], "import '../bloc/watchlist_bloc.dart';")
	assert.Contains(t, byPath[repoImplPath], "import '../../domain/repositories/watchlist_repository.dart';")
	assert.Contains(t, byPath[repoImplPath], "import '../models/watchlist_request.dart';")
	assert.Contains(t, byPath[usecasePath], "import '../../data/models/watchlist_response.dart';")
}

func TestGenerate_GroupedRoutesAndImports(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "grouped"

	files, err := gen.Generate("watchlist", "flutter_bloc_feature", &conv)
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	screenPath := "lib/features/watchlist/screen/watchlist_screen.dart"
	repoImplPath := "lib/features/watchlist/repository/watchlist_repository_impl.dart"

	require.Contains(t, byPath, screenPath)
	require.Contains(t, byPath, repoImplPath)
	assert.Contains(t, byPath[screenPath], "import '../bloc/watchlist_bloc.dart';")
	assert.Contains(t, byPath[repoImplPath], "import '../models/watchlist_request.dart';")
}

func TestGenerate_UnknownTemplate(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()

	_, err := gen.Generate("test", "nonexistent_template", &conv)
	assert.Error(t, err)
}

func TestWriteFiles_Basic(t *testing.T) {
	dir := t.TempDir()
	files := []GeneratedFile{
		{Path: "lib/features/test/test_bloc.dart", Content: "// bloc"},
		{Path: "lib/features/test/test_event.dart", Content: "// event"},
	}

	results, err := WriteFiles(files, dir, false)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.True(t, r.Written)
		assert.False(t, r.Skipped)
	}
}

func TestWriteFiles_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	files := []GeneratedFile{
		{Path: "lib/features/test/test_bloc.dart", Content: "// bloc"},
	}

	_, err := WriteFiles(files, dir, false)
	require.NoError(t, err)

	// Second write without force
	results, err := WriteFiles(files, dir, false)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Skipped)
	assert.Contains(t, results[0].Reason, "--force")
}

func TestWriteFiles_Force(t *testing.T) {
	dir := t.TempDir()
	files := []GeneratedFile{
		{Path: "lib/features/test/test_bloc.dart", Content: "// original"},
	}
	_, err := WriteFiles(files, dir, false)
	require.NoError(t, err)

	files[0].Content = "// updated"
	results, err := WriteFiles(files, dir, true)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Written)
}
