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

func TestGenerate_UsesRoleConventionsForNamesAndImports(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "grouped"
	conv.Roles["bloc"] = models.RoleConvention{FileSuffix: "controller", ClassSuffix: "Controller"}
	conv.Roles["event"] = models.RoleConvention{FileSuffix: "signal", ClassSuffix: "Signal"}
	conv.Roles["state"] = models.RoleConvention{FileSuffix: "snapshot", ClassSuffix: "Snapshot"}

	files, err := gen.GenerateWithOptions("fund_list", "flutter_bloc_feature", &conv, GenerateOptions{
		Roles: []string{"bloc", "event", "state"},
	})
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	blocPath := "lib/features/fund_list/bloc/fund_list_controller.dart"
	require.Contains(t, byPath, blocPath)
	assert.Contains(t, byPath[blocPath], "part 'fund_list_signal.dart';")
	assert.Contains(t, byPath[blocPath], "part 'fund_list_snapshot.dart';")
	assert.Contains(t, byPath[blocPath], "class FundListController extends Bloc<FundListSignal, FundListSnapshot>")
}

func TestGenerate_FlutterTemplatesUseScannedRoleClassNames(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "clean_architecture"
	conv.CommonImports = []string{
		"package:app/ui/material.dart",
		"package:app/state/custom_bloc.dart",
	}
	conv.Roles["request"] = models.RoleConvention{FileSuffix: "query", ClassSuffix: "Query"}
	conv.Roles["response"] = models.RoleConvention{FileSuffix: "result", ClassSuffix: "Result"}
	conv.Roles["repository"] = models.RoleConvention{FileSuffix: "gateway", ClassSuffix: "Gateway"}
	conv.Roles["repository_impl"] = models.RoleConvention{FileSuffix: "gateway_adapter", ClassSuffix: "GatewayAdapter"}
	conv.Roles["usecase"] = models.RoleConvention{FileSuffix: "interactor", ClassSuffix: "Interactor"}

	files, err := gen.GenerateWithOptions("fund_list", "flutter_bloc_feature", &conv, GenerateOptions{
		Roles: []string{"request", "response", "repository", "repository_impl", "usecase", "screen"},
	})
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	requestPath := "lib/features/fund_list/data/models/fund_list_query.dart"
	responsePath := "lib/features/fund_list/data/models/fund_list_result.dart"
	repoPath := "lib/features/fund_list/domain/repositories/fund_list_gateway.dart"
	repoImplPath := "lib/features/fund_list/data/repositories/fund_list_gateway_adapter.dart"
	usecasePath := "lib/features/fund_list/domain/usecases/fund_list_interactor.dart"
	screenPath := "lib/features/fund_list/presentation/screen/fund_list_screen.dart"

	require.Contains(t, byPath, requestPath)
	require.Contains(t, byPath, responsePath)
	require.Contains(t, byPath, repoPath)
	require.Contains(t, byPath, repoImplPath)
	require.Contains(t, byPath, usecasePath)
	assert.Contains(t, byPath[requestPath], "class FundListQuery")
	assert.Contains(t, byPath[responsePath], "class FundListResult")
	assert.Contains(t, byPath[repoPath], "abstract class FundListGateway")
	assert.Contains(t, byPath[repoPath], "Future<FundListResult> fetch(FundListQuery request);")
	assert.Contains(t, byPath[repoImplPath], "class FundListGatewayAdapter implements FundListGateway")
	assert.Contains(t, byPath[usecasePath], "class FundListInteractor")
	assert.Contains(t, byPath[usecasePath], "final FundListGateway _repository;")
	assert.Contains(t, byPath[screenPath], "import 'package:app/ui/material.dart';")
	assert.Contains(t, byPath[screenPath], "import 'package:app/state/custom_bloc.dart';")
}

func TestGenerate_GoTemplatesUseScannedRoleClassNames(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.ProjectType = "go"
	conv.FeatureRoot = "internal"
	conv.TestRoot = "internal"
	conv.FeatureStructure = "flat"
	conv.Roles = map[string]models.RoleConvention{
		"handler":         {FileSuffix: "endpoint", ClassSuffix: "Endpoint"},
		"service":         {FileSuffix: "workflow", ClassSuffix: "Workflow"},
		"repository":      {FileSuffix: "store", ClassSuffix: "Store"},
		"repository_impl": {FileSuffix: "store_impl", ClassSuffix: "StoreImpl"},
		"request":         {FileSuffix: "query", ClassSuffix: "Query"},
		"response":        {FileSuffix: "result", ClassSuffix: "Result"},
		"model":           {FileSuffix: "model", ClassSuffix: "Model"},
		"handler_test":    {FileSuffix: "endpoint_test", ClassSuffix: "EndpointTest"},
	}
	conv.PatternMappings = models.PatternMappings{
		"flat": {FileRoutes: map[string]string{
			"handler":         "",
			"service":         "",
			"repository":      "",
			"repository_impl": "",
			"model":           "",
			"handler_test":    "",
		}},
	}

	files, err := gen.Generate("payment", "go_clean_feature", &conv)
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	handlerPath := "internal/payment/payment_endpoint.go"
	modelPath := "internal/payment/payment_model.go"
	repoPath := "internal/payment/payment_store.go"
	repoImplPath := "internal/payment/payment_store_impl.go"
	servicePath := "internal/payment/payment_workflow.go"
	testPath := "internal/payment/payment_endpoint_test.go"

	require.Contains(t, byPath, handlerPath)
	require.Contains(t, byPath, modelPath)
	require.Contains(t, byPath, repoPath)
	require.Contains(t, byPath, repoImplPath)
	require.Contains(t, byPath, servicePath)
	require.Contains(t, byPath, testPath)
	assert.Contains(t, byPath[handlerPath], "type PaymentEndpoint struct")
	assert.Contains(t, byPath[handlerPath], "svc PaymentWorkflow")
	assert.Contains(t, byPath[handlerPath], "req := &PaymentQuery")
	assert.Contains(t, byPath[modelPath], "type PaymentQuery struct")
	assert.Contains(t, byPath[modelPath], "type PaymentResult struct")
	assert.Contains(t, byPath[repoPath], "type PaymentStore interface")
	assert.Contains(t, byPath[repoImplPath], "type PaymentStoreImpl struct")
	assert.Contains(t, byPath[servicePath], "type PaymentWorkflow interface")
	assert.Contains(t, byPath[testPath], "func TestPaymentEndpoint_Get")
}

func TestGenerate_SynthesizesDartRoleFromScannedAnatomy(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.FeatureStructure = "clean_architecture"
	conv.Roles["analytics"] = models.RoleConvention{FileSuffix: "analytics", ClassSuffix: "Analytics"}
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{
		{
			Name:      "existing",
			Path:      "lib/features/existing",
			Structure: "clean_architecture",
			FileRoutes: map[string]string{
				"analytics": "presentation/firebase",
			},
			Anatomy: map[string]models.FileAnatomy{
				"analytics": {
					Role:       "analytics",
					Path:       "lib/features/existing/presentation/firebase/existing_analytics.dart",
					ClassNames: []string{"ExistingAnalytics"},
					Types: []models.TypeAnatomy{
						{Name: "ExistingAnalytics", Kind: "class"},
					},
					Functions: []models.FunctionSignature{
						{Name: "trackScreenView", ReturnType: "void", Signature: "void trackScreenView()", IsMethod: true},
					},
				},
			},
		},
	}

	files, err := gen.GenerateWithOptions("fund_list", "flutter_bloc_feature", &conv, GenerateOptions{
		Roles:         []string{"analytics"},
		RolesExplicit: true,
	})
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Equal(t, "lib/features/fund_list/presentation/firebase/fund_list_analytics.dart", files[0].Path)
	assert.Contains(t, files[0].Content, "class FundListAnalytics")
	assert.Contains(t, files[0].Content, "const FundListAnalytics();")
	assert.Contains(t, files[0].Content, "void trackScreenView()")
}

func TestGenerate_SynthesizesGoRoleFromScannedAnatomy(t *testing.T) {
	gen := NewTemplateGenerator()
	conv := config.DefaultConventions()
	conv.ProjectType = "go"
	conv.FeatureRoot = "internal"
	conv.TestRoot = "internal"
	conv.FeatureStructure = "flat"
	conv.Roles = map[string]models.RoleConvention{
		"analytics": {FileSuffix: "analytics", ClassSuffix: "Analytics"},
	}
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{
		{
			Name:      "payment",
			Path:      "internal/payment",
			Structure: "flat",
			FileRoutes: map[string]string{
				"analytics": "",
			},
			Anatomy: map[string]models.FileAnatomy{
				"analytics": {
					Role:       "analytics",
					Path:       "internal/payment/payment_analytics.go",
					ClassNames: []string{"PaymentAnalytics"},
					Types: []models.TypeAnatomy{
						{Name: "PaymentAnalytics", Kind: "struct"},
					},
					Functions: []models.FunctionSignature{
						{Name: "Track", Receiver: "a *PaymentAnalytics", Params: []models.Parameter{{Name: "name", Type: "string"}}, IsMethod: true},
					},
				},
			},
		},
	}

	files, err := gen.GenerateWithOptions("refund", "go_clean_feature", &conv, GenerateOptions{
		Roles:         []string{"analytics"},
		RolesExplicit: true,
	})
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Equal(t, "internal/refund/refund_analytics.go", files[0].Path)
	assert.Contains(t, files[0].Content, "package refund")
	assert.Contains(t, files[0].Content, "type RefundAnalytics struct{}")
	assert.Contains(t, files[0].Content, "func (x *RefundAnalytics) Track(name string)")
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

func TestGenerateWithOptions_LikeRenameUsesScannedRoleFileTargets(t *testing.T) {
	gen := NewTemplateGenerator()
	baseDir := t.TempDir()
	writeFile := func(path, content string) {
		t.Helper()
		full := filepath.Join(baseDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	writeFile("internal/payment/repository/retirement.go", "package payment\n\ntype RetirementRepository struct{}\ntype RetirementAge int\n")
	writeFile("internal/payment/repository/retirement_plan_inputs.go", "package payment\n\ntype RetirementPlanInputs struct{}\n")

	conv := config.DefaultConventions()
	conv.ProjectType = "go"
	conv.FeatureRoot = "internal"
	conv.TestRoot = "internal"
	conv.FeatureStructure = "clean_architecture"
	conv.Roles = map[string]models.RoleConvention{
		"repository": {FileSuffix: "repository", ClassSuffix: "Repository"},
		"inputs":     {FileSuffix: "plan_inputs", ClassSuffix: "PlanInputs"},
	}
	conv.PatternMappings = models.PatternMappings{
		"clean_architecture": {FileRoutes: map[string]string{
			"repository": "repository",
			"inputs":     "repository",
		}},
	}
	like := models.FeatureAnalysis{
		Name:      "payment",
		Path:      "internal/payment",
		Structure: "clean_architecture",
		Files: map[string]string{
			"repository": "internal/payment/repository/retirement.go",
			"inputs":     "internal/payment/repository/retirement_plan_inputs.go",
		},
		FileRoutes: map[string]string{
			"repository": "repository",
			"inputs":     "repository",
		},
	}
	conv.FeaturesAnalysis.Features = []models.FeatureAnalysis{like}

	files, err := gen.GenerateWithOptions("new_feature", "go_clean_feature", &conv, GenerateOptions{
		Roles:         []string{"repository", "inputs"},
		RolesExplicit: true,
		LikeFeature:   &like,
		BaseDir:       baseDir,
	})
	require.NoError(t, err)

	byPath := make(map[string]string, len(files))
	for _, f := range files {
		byPath[f.Path] = f.Content
	}

	require.Contains(t, byPath, "internal/new_feature/repository/new_feature_repository.go")
	require.Contains(t, byPath, "internal/new_feature/repository/new_feature_plan_inputs.go")
	assert.Contains(t, byPath["internal/new_feature/repository/new_feature_repository.go"], "type NewFeatureRepository struct{}")
	assert.Contains(t, byPath["internal/new_feature/repository/new_feature_repository.go"], "type RetirementAge int")
	assert.Contains(t, byPath["internal/new_feature/repository/new_feature_plan_inputs.go"], "type NewFeaturePlanInputs struct{}")
}

func TestGenerateWithOptions_LikeFeatureUsesTemplateNotSource(t *testing.T) {
	// --like should generate clean template stubs for known roles (bloc, screen, etc.)
	// rather than copying source files. Base classes come from anatomy, not the source file.
	// Ancillary files that have no template are still copy-renamed from the source.
	gen := NewTemplateGenerator()
	baseDir := t.TempDir()

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
		Anatomy: map[string]models.FileAnatomy{
			"bloc": {
				Role:        "bloc",
				BaseClasses: []string{"BaseBlocScreen"},
			},
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
	// Base class comes from anatomy, not source file
	assert.Contains(t, files[0].Content, "class NewFeatureBloc extends BaseBlocScreen<NewFeatureEvent, NewFeatureState>")
	// Template generates clean stub — no fund_list business logic
	assert.NotContains(t, files[0].Content, "FundList")
	assert.NotContains(t, files[0].Content, "dashboard")
	// No usecase injection — "usecase" is not in the generated roles
	assert.NotContains(t, files[0].Content, "NewFeatureUseCase")
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
