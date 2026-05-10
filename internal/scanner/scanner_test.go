package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ── Project Detector ────────────────────────────────────────────────────────

func TestDetectProjectType_Flutter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte("flutter:\n  sdk: flutter\n"), 0o644))
	got, err := DetectProjectType(dir)
	require.NoError(t, err)
	assert.Equal(t, "flutter", got)
}

func TestDetectProjectType_Dart(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pubspec.yaml"), []byte("name: mylib\n"), 0o644))
	got, err := DetectProjectType(dir)
	require.NoError(t, err)
	assert.Equal(t, "dart", got)
}

func TestDetectProjectType_Go(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644))
	got, err := DetectProjectType(dir)
	require.NoError(t, err)
	assert.Equal(t, "go", got)
}

func TestDetectProjectType_Node(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644))
	got, err := DetectProjectType(dir)
	require.NoError(t, err)
	assert.Equal(t, "node", got)
}

func TestDetectProjectType_Unknown(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectProjectType(dir)
	require.NoError(t, err)
	assert.Equal(t, "unknown", got)
}

// ── Folder Scanner ───────────────────────────────────────────────────────────

func TestDetectFeatureRoot_Flutter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "lib/features"), 0o755))
	got, err := DetectFeatureRoot(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "lib/features", got)
}

func TestDetectFeatureRoot_FallsThrough(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "lib/modules"), 0o755))
	got, err := DetectFeatureRoot(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "lib/modules", got)
}

func TestDetectFeatureRoot_NotFound(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectFeatureRoot(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestDetectFeatureStructure_CleanArchitecture(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"home/presentation", "home/domain", "home/data"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}
	got, err := DetectFeatureStructure(dir)
	require.NoError(t, err)
	assert.Equal(t, "clean_architecture", got)
}

func TestDetectFeatureStructure_Grouped(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"home/bloc", "home/screen"} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, sub), 0o755))
	}
	got, err := DetectFeatureStructure(dir)
	require.NoError(t, err)
	assert.Equal(t, "grouped", got)
}

func TestDetectFeatureStructure_Flat(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "home"), 0o755))
	got, err := DetectFeatureStructure(dir)
	require.NoError(t, err)
	assert.Equal(t, "flat", got)
}

func TestDetectTestRoot_Flutter(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "test/features"), 0o755))
	got, err := DetectTestRoot(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "test/features", got)
}

// ── Naming Scanner ───────────────────────────────────────────────────────────

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestDetectNamingConventions_Screen(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "home/home_screen.dart"), "")
	writeFile(t, filepath.Join(dir, "profile/profile_screen.dart"), "")
	writeFile(t, filepath.Join(dir, "settings/settings_page.dart"), "")

	got, err := DetectNamingConventions(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "Screen", got.ScreenSuffix) // 2 vs 1
}

func TestDetectNamingConventions_BlocAndEvent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "home/home_bloc.dart"), "")
	writeFile(t, filepath.Join(dir, "home/home_event.dart"), "")
	writeFile(t, filepath.Join(dir, "home/home_state.dart"), "")

	got, err := DetectNamingConventions(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "Bloc", got.BlocSuffix)
	assert.Equal(t, "Event", got.EventSuffix)
	assert.Equal(t, "State", got.StateSuffix)
}

func TestDetectNamingConventions_Defaults_NonFlutter(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectNamingConventions(dir, "go")
	require.NoError(t, err)
	assert.Equal(t, "PascalCase", got.ClassCase)
	assert.Equal(t, "snake_case", got.FileCase)
}

// ── Import Scanner ───────────────────────────────────────────────────────────

func TestDetectCommonImports_Flutter(t *testing.T) {
	dir := t.TempDir()
	dartContent := `import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'dart:async';
`
	writeFile(t, filepath.Join(dir, "home/home_screen.dart"), dartContent)
	writeFile(t, filepath.Join(dir, "profile/profile_screen.dart"), dartContent)

	got, err := DetectCommonImports(dir, "flutter")
	require.NoError(t, err)
	assert.Contains(t, got, "package:flutter/material.dart")
	assert.Contains(t, got, "package:flutter_bloc/flutter_bloc.dart")
	// dart: core imports should be excluded
	for _, imp := range got {
		assert.False(t, len(imp) > 5 && imp[:5] == "dart:", "should exclude dart: imports")
	}
}

// ── Routing Scanner ──────────────────────────────────────────────────────────

const pubspecWithBloc = `
name: myapp
dependencies:
  flutter:
    sdk: flutter
  flutter_bloc: ^8.0.0
  go_router: ^12.0.0
flutter:
  uses-material-design: true
`

func TestDetectStateManagement_Bloc(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), pubspecWithBloc)
	got, err := DetectStateManagement(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "flutter_bloc", got)
}

func TestDetectStateManagement_Unknown(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectStateManagement(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "unknown", got)
}

func TestDetectRouting_GoRouter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), pubspecWithBloc)
	writeFile(t, filepath.Join(dir, "lib/router/app_router.dart"), "")

	got, err := DetectRouting(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "go_router", got.Type)
	assert.Equal(t, "lib/router/app_router.dart", got.RouteFile)
}

// ── Flutter Scanner (orchestrator) ──────────────────────────────────────────

func buildFlutterProject(t *testing.T, dir string) {
	t.Helper()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), pubspecWithBloc)
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
		writeFile(t, filepath.Join(dir, f), "import 'package:flutter_bloc/flutter_bloc.dart';\n")
	}
}

func TestFlutterScanner_Scan(t *testing.T) {
	dir := t.TempDir()
	buildFlutterProject(t, dir)

	s := &FlutterScanner{}
	conv, err := s.Scan(dir)
	require.NoError(t, err)
	require.NotNil(t, conv)

	assert.Equal(t, "flutter", conv.ProjectType)
	assert.Equal(t, "lib/features", conv.FeatureRoot)
	assert.Equal(t, "grouped", conv.FeatureStructure)
	assert.Equal(t, "test/features", conv.TestRoot)
	assert.Equal(t, "flutter_bloc", conv.StateManagement)
	assert.Equal(t, "go_router", conv.Routing.Type)
	assert.Equal(t, "lib/router/app_router.dart", conv.Routing.RouteFile)
	assert.Equal(t, "Screen", conv.Naming.ScreenSuffix)
	assert.Equal(t, "Bloc", conv.Naming.BlocSuffix)
	assert.Equal(t, "Repository", conv.Naming.RepositorySuffix)
	require.Len(t, conv.FeaturesAnalysis.Features, 2)
	assert.Equal(t, 1, conv.FeaturesAnalysis.PatternDistribution["clean_architecture"].Count)
	assert.Equal(t, 1, conv.FeaturesAnalysis.PatternDistribution["grouped"].Count)
	assert.Equal(t, "grouped", conv.FeaturesAnalysis.RecommendedPattern)
	assert.Equal(t, "domain/repository", conv.PatternMappings["clean_architecture"].FileRoutes["repository"])
	assert.Equal(t, "presentation/bloc", conv.PatternMappings["clean_architecture"].FileRoutes["bloc"])
}

func TestAnalyzeFeatureRoot_DetectsNestedFlowFeature(t *testing.T) {
	dir := t.TempDir()
	blocContent := `import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:investment_module/core/base/bloc/base_event.dart';

class FundListBloc extends BaseBloc<FundListEvent, FundListState> {
  FundListBloc(GetFundListUseCase getFundListUseCase, FundListAnalytics analytics);

  Future<void> _onStarted(FundListStarted event, Emitter<FundListState> emit) async {}
}
`
	screenContent := `import 'package:flutter/material.dart';
import 'package:investment_module/core/base/bloc/base_bloc_screen.dart';

class FundListScreen extends BaseBlocScreen<FundListBloc, FundListState> {
  @override
  FundListBloc createBloc(BuildContext context) => FundListBloc();

  @override
  Widget buildContent(BuildContext context, FundListState state) => const SizedBox();
}
`
	analyticsContent := `class FundListAnalytics {
  void trackScreenView() {}
}
`
	files := []string{
		"lib/features/investment/fund_list/presentation/bloc/fund_list_event.dart",
		"lib/features/investment/fund_list/presentation/bloc/fund_list_state.dart",
		"lib/features/investment/fund_list/domain/repositories/fund_list_repository.dart",
		"lib/features/investment/fund_list/data/repositories/fund_list_repository_impl.dart",
		"lib/features/investment/fund_detail/fund_detail_screen.dart",
	}
	for _, f := range files {
		writeFile(t, filepath.Join(dir, f), "")
	}
	writeFile(t, filepath.Join(dir, "lib/features/investment/fund_list/presentation/screen/fund_list_screen.dart"), screenContent)
	writeFile(t, filepath.Join(dir, "lib/features/investment/fund_list/presentation/bloc/fund_list_bloc.dart"), blocContent)
	writeFile(t, filepath.Join(dir, "lib/features/investment/fund_list/presentation/firebase/fund_list_screen_analytics.dart"), analyticsContent)

	analysis, err := AnalyzeFeatureRoot(dir, "lib/features")
	require.NoError(t, err)
	require.Len(t, analysis.Features, 2)

	byPath := map[string]models.FeatureAnalysis{}
	for _, feature := range analysis.Features {
		byPath[feature.Path] = feature
	}

	fundList := byPath["lib/features/investment/fund_list"]
	assert.Equal(t, "fund_list", fundList.Name)
	assert.Equal(t, "investment", fundList.Parent)
	assert.Equal(t, 2, fundList.Depth)
	assert.Equal(t, "clean_architecture", fundList.Structure)
	assert.Equal(t, "lib/features/investment/fund_list/presentation/bloc/fund_list_bloc.dart", fundList.Files["bloc"])
	assert.Equal(t, "presentation/bloc", fundList.FileRoutes["bloc"])
	assert.Equal(t, "presentation/screen", fundList.FileRoutes["screen"])
	assert.NotContains(t, byPath, "lib/features/investment")
	assert.NotContains(t, byPath, "lib/features/investment/fund_list/presentation/firebase")

	blocAnatomy := fundList.Anatomy["bloc"]
	assert.Contains(t, blocAnatomy.BaseClasses, "BaseBloc")
	assert.Contains(t, blocAnatomy.Methods, "_onStarted")
	assert.Contains(t, blocAnatomy.ConstructorDeps, "GetFundListUseCase")
	assert.Contains(t, blocAnatomy.ConstructorDeps, "FundListAnalytics")
	assert.NotContains(t, blocAnatomy.ConstructorDeps, "FundListBloc")

	screenAnatomy := fundList.Anatomy["screen"]
	assert.Contains(t, screenAnatomy.BaseClasses, "BaseBlocScreen")
	assert.Contains(t, screenAnatomy.Methods, "createBloc")
	assert.Contains(t, screenAnatomy.Methods, "buildContent")

	require.Contains(t, analysis.RoleAnatomy, "screen")
	assert.Equal(t, 1, analysis.RoleAnatomy["screen"].FeatureCount)
	assert.Equal(t, "BaseBlocScreen", analysis.RoleAnatomy["screen"].BaseClasses[0].Name)
	assert.Equal(t, "buildContent", analysis.RoleAnatomy["screen"].Methods[0].Name)
}

func TestAnalyzeFeatureRoot_DiscoversGenericRoleFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "lib/features/fund_list/fund_list_bloc.dart"), "class FundListBloc {}\n")
	writeFile(t, filepath.Join(dir, "lib/features/fund_list/data/models/fund_list_request.dart"), "class FundListRequest {}\n")
	writeFile(t, filepath.Join(dir, "lib/features/fund_list/data/models/fund_list_response.dart"), "class FundListResponse {}\n")
	writeFile(t, filepath.Join(dir, "lib/features/fund_list/data/models/fund_list_payload.dart"), "class FundListPayload {}\n")

	naming := models.NamingConvention{
		BlocSuffix: "Bloc",
		SuffixRoles: map[string]string{
			"bloc": "bloc",
		},
	}
	analysis, err := AnalyzeFeatureRoot(dir, "lib/features", naming)
	require.NoError(t, err)
	require.Len(t, analysis.Features, 1)

	feature := analysis.Features[0]
	assert.Equal(t, "lib/features/fund_list/data/models/fund_list_request.dart", feature.Files["request"])
	assert.Equal(t, "lib/features/fund_list/data/models/fund_list_response.dart", feature.Files["response"])
	assert.Equal(t, "lib/features/fund_list/data/models/fund_list_payload.dart", feature.Files["payload"])
	assert.Equal(t, "data/models", feature.FileRoutes["payload"])

	roles := InferRoleConventions(analysis.Features, naming)
	assert.Equal(t, models.RoleConvention{FileSuffix: "request", ClassSuffix: "Request"}, roles["request"])
	assert.Equal(t, models.RoleConvention{FileSuffix: "response", ClassSuffix: "Response"}, roles["response"])
	assert.Equal(t, models.RoleConvention{FileSuffix: "payload", ClassSuffix: "Payload"}, roles["payload"])
}

func TestFlutterScanner_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), "flutter:\n  sdk: flutter\n")

	s := &FlutterScanner{}
	conv, err := s.Scan(dir)
	require.NoError(t, err)
	assert.Equal(t, "flutter", conv.ProjectType)
	assert.Equal(t, "lib/features", conv.FeatureRoot)
}

func TestGoScanner_MinimalDir(t *testing.T) {
	s := &GoScanner{}
	conv, err := s.Scan(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, "go", conv.ProjectType)
	assert.Equal(t, "internal", conv.FeatureRoot)
}

func TestDetectNamingConventions_Repository(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "home/home_repository.dart"), "")
	writeFile(t, filepath.Join(dir, "profile/profile_repository.dart"), "")

	got, err := DetectNamingConventions(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "Repository", got.RepositorySuffix)
}

func TestMajorityVote(t *testing.T) {
	votes := map[string]int{"Screen": 8, "Page": 2}
	assert.Equal(t, "Screen", majorityVote(votes, "Screen"))

	empty := map[string]int{"Screen": 0}
	assert.Equal(t, "Screen", majorityVote(empty, "Screen"))
}

func TestDetectCommonImports_Empty(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectCommonImports(dir, "flutter")
	require.NoError(t, err)
	assert.Empty(t, got)
}

// models.NamingConvention is used via type assertion; ensure interface satisfied
var _ Scanner = (*FlutterScanner)(nil)
var _ Scanner = (*GoScanner)(nil)

func TestGoScanner_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/myapp\n\ngo 1.21\n")

	s := &GoScanner{}
	conv, err := s.Scan(dir)
	require.NoError(t, err)
	assert.Equal(t, "go", conv.ProjectType)
	assert.Equal(t, "example.com/myapp", conv.ModulePath)
	assert.Equal(t, "net/http", conv.Routing.Type)
}

func TestDetectNamingConventions_Usecase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "home/get_home_usecase.dart"), "")
	writeFile(t, filepath.Join(dir, "profile/get_profile_use_case.dart"), "")

	got, err := DetectNamingConventions(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "UseCase", got.UsecaseSuffix)
}

// Test ignored project types for state detection
func TestDetectStateManagement_Go(t *testing.T) {
	dir := t.TempDir()
	got, err := DetectStateManagement(dir, "go")
	require.NoError(t, err)
	assert.Equal(t, "unknown", got)
}

// Riverpod detection
func TestDetectStateManagement_Riverpod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), `
name: myapp
dependencies:
  flutter:
    sdk: flutter
  flutter_riverpod: ^2.0.0
`)
	got, err := DetectStateManagement(dir, "flutter")
	require.NoError(t, err)
	assert.Equal(t, "riverpod", got)
}

func TestConventionComplete(t *testing.T) {
	// Ensure models.Convention is compatible with what scanner returns
	var _ *models.Convention = &models.Convention{}
}
