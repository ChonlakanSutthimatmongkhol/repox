package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/generator"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/retriever"
)

var (
	generateForce        bool
	generateDryRun       bool
	generateTemplate     string
	generatePattern      string
	generateWithExamples bool
	generateRoles        string
	generateLike         string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate scaffolds for features",
}

var generateFeatureCmd = &cobra.Command{
	Use:   "feature <name-or-path>",
	Short: "Generate a feature scaffold",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenerateFeature,
}

func init() {
	generateFeatureCmd.Flags().BoolVarP(&generateForce, "force", "f", false, "Overwrite existing files")
	generateFeatureCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Preview files without writing")
	generateFeatureCmd.Flags().StringVarP(&generateTemplate, "template", "t", "", "Template to use (overrides config)")
	generateFeatureCmd.Flags().StringVar(&generatePattern, "pattern", "", "Feature pattern to use (flat, grouped, clean_architecture)")
	generateFeatureCmd.Flags().BoolVar(&generateWithExamples, "with-examples", false, "Find and show similar existing features before generating")
	generateFeatureCmd.Flags().StringVar(&generateRoles, "roles", "", "Comma-separated file roles to generate (e.g. bloc,event,state,screen)")
	generateFeatureCmd.Flags().StringVar(&generateLike, "like", "", "Use an existing feature path/name as the shape reference")
	generateCmd.AddCommand(generateFeatureCmd)
	rootCmd.AddCommand(generateCmd)
}

func runGenerateFeature(cmd *cobra.Command, args []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	featureName := args[0]

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("generate: getwd: %w", err)
	}

	cfg, err := config.Load[models.Config](config.RepoxPath("config.json"))
	if err != nil {
		return fmt.Errorf("generate: load config: %w", err)
	}

	conv, err := config.Load[models.Convention](config.RepoxPath("conventions.json"))
	if err != nil {
		return fmt.Errorf("generate: load conventions: %w", err)
	}
	if err := applyGeneratePattern(&conv); err != nil {
		return err
	}
	tmplName := cfg.DefaultTemplate
	if generateTemplate != "" {
		tmplName = generateTemplate
	}

	roles := parseRoles(generateRoles)
	var likeFeature *models.FeatureAnalysis
	if strings.TrimSpace(generateLike) != "" {
		feature, ok := findFeatureAnalysis(&conv, generateLike)
		if ok {
			likeFeature = &feature
		}
	}
	if err := applyGenerateLike(&conv, featureName, generateLike, tmplName, &roles); err != nil {
		return err
	}

	// --with-examples: show similar features found
	if generateWithExamples {
		examples, err := config.Load[[]models.Example](config.RepoxPath("examples.json"))
		if err != nil || len(examples) == 0 {
			examples, _ = retriever.IndexFeatures(cwd, &conv)
		}
		similar := retriever.FindSimilar(featureName, examples, 3)
		if len(similar) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Similar features found:")
			for _, ex := range similar {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%s)\n", ex.Name, ex.Path)
			}
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	// Generate files from local templates.
	genMode := "template"
	if likeFeature != nil {
		genMode = "like"
	}
	gen := generator.NewTemplateGenerator()
	files, err := gen.GenerateWithOptions(featureName, tmplName, &conv, generator.GenerateOptions{
		Roles:         roles,
		RolesExplicit: strings.TrimSpace(generateRoles) != "",
		LikeFeature:   likeFeature,
		BaseDir:       cwd,
	})
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if generateDryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry run — no files written:")
		for _, f := range files {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", f.Path)
		}
		return nil
	}

	results, err := generator.WriteFiles(files, cwd, generateForce)
	if err != nil {
		return fmt.Errorf("generate: write files: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	written, skipped := 0, 0
	var writtenPaths []string
	for _, r := range results {
		if r.Written {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", green("created"), r.Path)
			writtenPaths = append(writtenPaths, filepath.Join(cwd, r.Path))
			written++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", yellow("skipped"), r.Reason)
			skipped++
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d created, %d skipped\n", written, skipped)

	if written > 0 && !generateDryRun {
		printChecklist(cmd, featureName, files, &conv)
	}

	// Run formatter on written files
	runFormatter(writtenPaths, cmd)

	// Log generation (and save snapshot for learner)
	genID := fmt.Sprintf("gen_%d", time.Now().Unix())
	filePaths := make([]string, len(results))
	for i, r := range results {
		filePaths[i] = r.Path
	}

	snapshotDir := saveSnapshot(genID, files, cwd)

	_ = appendGeneration(models.Generation{
		ID:          genID,
		FeatureName: featureName,
		Template:    tmplName,
		Mode:        genMode,
		Files:       filePaths,
		SnapshotDir: snapshotDir,
		CreatedAt:   time.Now(),
	})

	return nil
}

func applyGeneratePattern(conv *models.Convention) error {
	pattern := recommendedPatternForGeneration(conv)
	if generatePattern != "" {
		pattern = generatePattern
	}
	if pattern == "" {
		pattern = "flat"
	}
	if !validPattern(pattern, conv) {
		return fmt.Errorf("generate: unsupported pattern %q (not found in .repox/conventions.json pattern_mappings)", pattern)
	}
	conv.FeatureStructure = pattern
	return nil
}

func applyRecommendedPattern(conv *models.Convention) {
	pattern := recommendedPatternForGeneration(conv)
	if pattern == "" {
		pattern = "flat"
	}
	conv.FeatureStructure = pattern
}

func recommendedPatternForGeneration(conv *models.Convention) string {
	if conv.FeaturesAnalysis.RecommendedPattern != "" {
		return conv.FeaturesAnalysis.RecommendedPattern
	}
	return conv.FeatureStructure
}

func validPattern(pattern string, conv *models.Convention) bool {
	if pattern == "" {
		return false
	}
	if conv.PatternMappings == nil {
		return true
	}
	_, ok := conv.PatternMappings[pattern]
	return ok
}

func parseRoles(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	seen := map[string]bool{}
	var roles []string
	for _, part := range strings.Split(value, ",") {
		role := strings.TrimSpace(part)
		if role == "" || seen[role] {
			continue
		}
		seen[role] = true
		roles = append(roles, role)
	}
	return roles
}

func applyGenerateLike(conv *models.Convention, targetFeature, like, templateName string, roles *[]string) error {
	if strings.TrimSpace(like) == "" {
		return nil
	}
	feature, ok := findFeatureAnalysis(conv, like)
	if !ok {
		return fmt.Errorf("generate: --like feature %q not found in scanned conventions; run `repox scan` or choose an existing feature path", like)
	}

	targetPath := normalizeFeaturePathForCLI(targetFeature)
	liked := feature
	liked.Path = filepath.ToSlash(filepath.Join(conv.FeatureRoot, targetPath))
	liked.Parent = filepath.ToSlash(filepath.Dir(targetPath))
	if liked.Parent == "." {
		liked.Parent = ""
	}
	liked.Name = filepath.Base(targetPath)
	conv.FeaturesAnalysis.Features = append(conv.FeaturesAnalysis.Features, liked)
	if liked.Structure != "" {
		conv.FeatureStructure = liked.Structure
	}
	if len(*roles) == 0 {
		templateRoles, err := generator.TemplateRoles(templateName)
		if err != nil {
			return err
		}
		*roles = rolesForLikeFeature(feature, conv, templateRoles)
	}
	return nil
}

func findFeatureAnalysis(conv *models.Convention, query string) (models.FeatureAnalysis, bool) {
	query = filepath.ToSlash(strings.Trim(query, "/ "))
	queryBase := filepath.Base(query)
	for _, feature := range conv.FeaturesAnalysis.Features {
		rel := featurePathWithoutRootForCLI(conv.FeatureRoot, feature.Path)
		if feature.Path == query || rel == query || feature.Name == query || feature.Name == queryBase {
			return feature, true
		}
	}
	return models.FeatureAnalysis{}, false
}

func featureRolesFromAnalysis(feature models.FeatureAnalysis) []string {
	roles := make([]string, 0, len(feature.Files))
	for role := range feature.Files {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func rolesForLikeFeature(feature models.FeatureAnalysis, conv *models.Convention, templateRoles []string) []string {
	seen := map[string]bool{}
	var roles []string
	for _, role := range featureRolesFromAnalysis(feature) {
		seen[role] = true
		roles = append(roles, role)
	}
	for _, role := range scannedTemplateRoles(feature.Structure, conv, templateRoles) {
		if isTestRole(role) {
			continue
		}
		if seen[role] {
			continue
		}
		seen[role] = true
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func scannedTemplateRoles(pattern string, conv *models.Convention, templateRoles []string) []string {
	templateRoleSet := map[string]bool{}
	for _, role := range templateRoles {
		if role != "" && !isTestRole(role) {
			templateRoleSet[role] = true
		}
	}

	seen := map[string]bool{}
	for _, feature := range conv.FeaturesAnalysis.Features {
		if pattern != "" && feature.Structure != pattern {
			continue
		}
		for role := range feature.Files {
			if templateRoleSet[role] {
				seen[role] = true
			}
		}
	}
	for role := range conv.FeaturesAnalysis.RoleAnatomy {
		if templateRoleSet[role] {
			seen[role] = true
		}
	}

	if len(seen) == 0 {
		for role := range templateRoleSet {
			seen[role] = true
		}
	}
	roles := make([]string, 0, len(seen))
	for role := range seen {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func isTestRole(role string) bool {
	return strings.HasSuffix(role, "_test") || role == "test"
}

func normalizeFeaturePathForCLI(featureName string) string {
	parts := strings.FieldsFunc(featureName, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		normalized = append(normalized, generator.ToSnakeCase(part))
	}
	if len(normalized) == 0 {
		return generator.ToSnakeCase(featureName)
	}
	return filepath.Join(normalized...)
}

func featurePathWithoutRootForCLI(featureRoot, path string) string {
	featureRoot = filepath.ToSlash(strings.Trim(featureRoot, "/"))
	path = filepath.ToSlash(strings.Trim(path, "/"))
	if featureRoot != "" && strings.HasPrefix(path, featureRoot+"/") {
		return strings.TrimPrefix(path, featureRoot+"/")
	}
	return path
}

// saveSnapshot copies generated file contents to .repox/snapshots/<genID>/ for later diff.
func saveSnapshot(genID string, files []generator.GeneratedFile, baseDir string) string {
	snapDir := config.RepoxPath("snapshots/" + genID)
	for _, f := range files {
		dest := filepath.Join(snapDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}
		_ = os.WriteFile(dest, []byte(f.Content), 0o644)
	}
	// Return absolute path so diff_reader can locate it from any working directory.
	abs, err := filepath.Abs(snapDir)
	if err != nil {
		return snapDir
	}
	return abs
}

func appendGeneration(gen models.Generation) error {
	existing, err := config.Load[[]models.Generation](config.RepoxPath("generations.json"))
	if err != nil {
		existing = []models.Generation{}
	}
	existing = append(existing, gen)
	return config.Save(config.RepoxPath("generations.json"), existing)
}

func runFormatter(paths []string, cmd *cobra.Command) {
	if len(paths) == 0 {
		return
	}
	var dartFiles, goFiles []string
	for _, p := range paths {
		switch filepath.Ext(p) {
		case ".dart":
			dartFiles = append(dartFiles, p)
		case ".go":
			goFiles = append(goFiles, p)
		}
	}
	if len(dartFiles) > 0 {
		runDartFormat(dartFiles, cmd)
	}
	if len(goFiles) > 0 {
		runGoFormat(goFiles, cmd)
	}
}

func runDartFormat(files []string, cmd *cobra.Command) {
	dartPath, err := exec.LookPath("dart")
	if err != nil {
		return
	}
	args := append([]string{"format"}, files...)
	out, err := exec.Command(dartPath, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: dart format failed: %s\n", string(out))
	}
}

func runGoFormat(files []string, cmd *cobra.Command) {
	gofmtPath, err := exec.LookPath("gofmt")
	if err != nil {
		return
	}
	args := append([]string{"-w"}, files...)
	out, err := exec.Command(gofmtPath, args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: gofmt failed: %s\n", string(out))
	}
}

func printChecklist(cmd *cobra.Command, featureName string, files []generator.GeneratedFile, conv *models.Convention) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	roleSet := map[string]string{}
	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f.Path), ".dart")
		base = strings.TrimSuffix(base, ".go")
		for _, role := range []string{"bloc", "usecase", "repository", "repository_impl", "screen", "request", "response"} {
			if strings.HasSuffix(base, "_"+role) || strings.HasSuffix(base, "_"+strings.ReplaceAll(role, "_", "")) {
				roleSet[role] = f.Path
				break
			}
		}
	}

	leaf := filepath.Base(normalizeFeaturePathForCLI(featureName))
	pascal := generator.ToPascalCase(leaf)
	snake := generator.ToSnakeCase(leaf)
	usecaseSuffix := conv.Naming.UsecaseSuffix
	repoSuffix := conv.Naming.RepositorySuffix

	fmt.Fprintln(cmd.OutOrStdout(), "\n"+cyan("Next steps:"))

	step := 1
	if _, hasBlocFile := roleSet["bloc"]; hasBlocFile {
		if _, hasUsecase := roleSet["usecase"]; hasUsecase {
			fmt.Fprintf(cmd.OutOrStdout(), "  %d. Register in service locator:\n", step)
			fmt.Fprintf(cmd.OutOrStdout(), "       %s\n", dim("sl.registerFactory(() => "+pascal+usecaseSuffix+"(sl()))"))
			fmt.Fprintf(cmd.OutOrStdout(), "       %s\n", dim("sl.registerFactory(() => "+pascal+"Bloc(sl()))"))
			step++
		}
	}
	if _, hasRepo := roleSet["repository_impl"]; hasRepo {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. Bind repository in service locator:\n", step)
		fmt.Fprintf(cmd.OutOrStdout(), "       %s\n", dim("sl.registerLazySingleton<"+pascal+repoSuffix+">(() => "+pascal+repoSuffix+"Impl(sl()))"))
		step++
	}
	if _, hasScreen := roleSet["screen"]; hasScreen {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. Add route in router:\n", step)
		fmt.Fprintf(cmd.OutOrStdout(), "       %s\n", dim("GoRoute(path: '/"+snake+"', builder: (_, __) => BlocProvider(create: (_) => sl<"+pascal+"Bloc>(), child: const "+pascal+"Screen()))"))
		step++
	}
	if _, hasRepo := roleSet["repository_impl"]; hasRepo {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. Implement fetch() in %sRepositoryImpl\n", step, pascal)
		step++
	}
	_ = step
}
