package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
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

func TestGenerate_NestedFeaturePathUsesLeafNaming(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "clean_architecture"

	files, err := gen.Generate("investment/fund_list", "flutter_bloc_feature", &conv)
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	screenPath := "lib/features/investment/fund_list/presentation/screen/fund_list_screen.dart"
	blocPath := "lib/features/investment/fund_list/presentation/bloc/fund_list_bloc.dart"
	testPath := "test/features/investment/fund_list/fund_list_bloc_test.dart"

	require.Contains(t, byPath, screenPath)
	require.Contains(t, byPath, blocPath)
	require.Contains(t, byPath, testPath)
	assert.Contains(t, byPath[blocPath], "class FundListBloc")
	assert.NotContains(t, byPath[blocPath], "InvestmentFundListBloc")
	assert.Contains(t, byPath[screenPath], "import '../bloc/fund_list_bloc.dart';")
}

func TestGenerate_UsesScannedNestedFeatureRoutes(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "flat"
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{
		{
			Name:      "fund_list",
			Path:      "lib/features/investment/fund_list",
			Parent:    "investment",
			Structure: "clean_architecture",
			FileRoutes: map[string]string{
				"bloc":   "presentation",
				"event":  "presentation",
				"screen": "presentation",
				"state":  "presentation",
			},
		},
	}

	files, err := gen.Generate("investment/fund_list", "flutter_bloc_feature", &conv)
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	blocPath := "lib/features/investment/fund_list/presentation/fund_list_bloc.dart"
	repoPath := "lib/features/investment/fund_list/domain/repositories/fund_list_repository.dart"
	repoImplPath := "lib/features/investment/fund_list/data/repositories/fund_list_repository_impl.dart"

	require.Contains(t, byPath, blocPath)
	require.Contains(t, byPath, repoPath)
	require.Contains(t, byPath, repoImplPath)
	assert.Contains(t, byPath[repoImplPath], "import '../../domain/repositories/fund_list_repository.dart';")
}

func TestGenerateWithOptions_FiltersRoles(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "clean_architecture"

	files, err := gen.GenerateWithOptions("investment/fund_list", "flutter_bloc_feature", &conv, GenerateOptions{
		Roles: []string{"bloc", "event", "state", "screen"},
	})
	require.NoError(t, err)

	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}

	assert.Len(t, files, 4)
	assert.Contains(t, paths, "lib/features/investment/fund_list/presentation/bloc/fund_list_bloc.dart")
	assert.Contains(t, paths, "lib/features/investment/fund_list/presentation/screen/fund_list_screen.dart")
	for _, path := range paths {
		assert.NotContains(t, path, "repository")
		assert.NotContains(t, path, "test")
	}
}

func TestGenerateWithOptions_LikeFeatureRewritesSourceFiles(t *testing.T) {
	gen := NewTemplateGenerator()
	baseDir := t.TempDir()
	sourcePath := filepath.Join(baseDir, "lib/features/investment/fund_list/presentation/fund_list_bloc.dart")
	require.NoError(t, os.MkdirAll(filepath.Dir(sourcePath), 0o755))
	require.NoError(t, os.WriteFile(sourcePath, []byte(`import 'package:investment_module/features/investment/fund_list/presentation/fund_list_event.dart';
import 'package:investment_module/common/widgets/shared_fund_list_item/shared_fund_list_item_widget.dart';
import 'package:investment_module/features/investment/dashboard/domain/usecase/get_highlight_fund_details_usecase.dart';

class FundListBloc extends BaseBlocScreen<FundListEvent, FundListState> {
  FundListBloc() : super(FundListInitialState());

  BaseTrackingLandedEvent createBaseTrackingLandedEvent() {
    return FundListTrackLandingEvent();
  }

  SharedFundListItemWidget buildSharedWidget() {
    return SharedFundListItemWidget();
  }
}
`), 0o644))

	conv := config.DefaultConventions()
	conv.FeatureStructure = "flat"
	like := models.FeatureAnalysis{
		Name:      "fund_list",
		Path:      "lib/features/investment/fund_list",
		Structure: "clean_architecture",
		Files: map[string]string{
			"bloc": "lib/features/investment/fund_list/presentation/fund_list_bloc.dart",
		},
		FileRoutes: map[string]string{
			"bloc": "presentation",
		},
	}
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{like}

	files, err := gen.GenerateWithOptions("investment/new_feature", "flutter_bloc_feature", &conv, GenerateOptions{
		Roles:       []string{"bloc"},
		LikeFeature: &like,
		BaseDir:     baseDir,
	})
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Equal(t, "lib/features/investment/new_feature/presentation/new_feature_bloc.dart", files[0].Path)
	// Source file identifiers are renamed
	assert.Contains(t, files[0].Content, "class NewFeatureBloc extends BaseBlocScreen<NewFeatureEvent, NewFeatureState>")
	assert.Contains(t, files[0].Content, "return NewFeatureTrackLandingEvent();")
	// Own-feature import is renamed
	assert.Contains(t, files[0].Content, "investment/new_feature/presentation/new_feature_event.dart")
	// Shared (non-feature) import is preserved
	assert.Contains(t, files[0].Content, "common/widgets/shared_fund_list_item/shared_fund_list_item_widget.dart")
	assert.Contains(t, files[0].Content, "SharedFundListItemWidget")
	// Cross-feature import from dashboard is stripped
	assert.NotContains(t, files[0].Content, "dashboard")
	assert.NotContains(t, files[0].Content, "GetHighlightFundDetailsUseCase")
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
