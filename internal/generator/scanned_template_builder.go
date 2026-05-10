package generator

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/conventions"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

// ScannedTemplateBuilder synthesizes role files when no embedded .tmpl exists.
type ScannedTemplateBuilder struct {
	conv    *models.Convention
	profile *conventions.ProjectProfile
}

func NewScannedTemplateBuilder(conv *models.Convention, profile *conventions.ProjectProfile) *ScannedTemplateBuilder {
	return &ScannedTemplateBuilder{conv: conv, profile: profile}
}

func (b *ScannedTemplateBuilder) Build(ctx TemplateContext, target conventions.TargetFeature, role string) (GeneratedFile, bool) {
	anatomy, ok := b.findRoleAnatomy(role)
	if !ok {
		return GeneratedFile{}, false
	}
	ext := filepath.Ext(anatomy.Path)
	if ext == "" {
		if b.conv.ProjectType == "go" {
			ext = ".go"
		} else {
			ext = ".dart"
		}
	}
	path := b.profile.FeatureFilePath(target, role, ext)
	content := ""
	switch ext {
	case ".dart":
		content = b.buildDart(ctx, role, anatomy)
	case ".go":
		content = b.buildGo(ctx, role, anatomy)
	default:
		return GeneratedFile{}, false
	}
	return GeneratedFile{Path: path, Content: content}, strings.TrimSpace(content) != ""
}

func (b *ScannedTemplateBuilder) findRoleAnatomy(role string) (models.FileAnatomy, bool) {
	for _, feature := range b.conv.FeaturesAnalysis.Features {
		if feature.Structure != "" && b.conv.FeatureStructure != "" && feature.Structure != b.conv.FeatureStructure {
			continue
		}
		if anatomy, ok := feature.Anatomy[role]; ok {
			return anatomy, true
		}
	}
	for _, feature := range b.conv.FeaturesAnalysis.Features {
		if anatomy, ok := feature.Anatomy[role]; ok {
			return anatomy, true
		}
	}
	return models.FileAnatomy{}, false
}

func (b *ScannedTemplateBuilder) buildDart(ctx TemplateContext, role string, anatomy models.FileAnatomy) string {
	className := ctx.Class(role)
	base := ""
	if len(anatomy.BaseClasses) > 0 {
		base = " extends " + anatomy.BaseClasses[0]
	}
	var out strings.Builder
	for _, imp := range nonFeatureImports(anatomy.Imports) {
		fmt.Fprintf(&out, "import '%s';\n", imp)
	}
	if len(anatomy.Imports) > 0 {
		out.WriteString("\n")
	}
	fmt.Fprintf(&out, "class %s%s {\n", className, base)
	fmt.Fprintf(&out, "  const %s();\n", className)
	for _, fn := range anatomy.Functions {
		if fn.Name == "" || startsWithUnderscore(fn.Name) {
			continue
		}
		out.WriteString("\n")
		fmt.Fprintf(&out, "  %s %s(%s) {\n", dartReturnType(fn), fn.Name, dartParams(fn.Params))
		if ret := dartReturnStub(fn.ReturnType); ret != "" {
			fmt.Fprintf(&out, "    %s\n", ret)
		}
		out.WriteString("  }\n")
	}
	out.WriteString("}\n")
	return out.String()
}

func (b *ScannedTemplateBuilder) buildGo(ctx TemplateContext, role string, anatomy models.FileAnatomy) string {
	className := ctx.Class(role)
	packageName := ctx.PackageName
	if strings.Contains(role, "test") {
		packageName += "_test"
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	imports := goImportsForFunctions(anatomy)
	if len(imports) > 0 {
		out.WriteString("import (\n")
		for _, imp := range imports {
			fmt.Fprintf(&out, "\t%q\n", imp)
		}
		out.WriteString(")\n\n")
	}
	kind := "struct"
	if len(anatomy.Types) > 0 && anatomy.Types[0].Kind == "interface" {
		kind = "interface"
	}
	if kind == "interface" {
		fmt.Fprintf(&out, "type %s interface {\n", className)
		for _, fn := range anatomy.Functions {
			if fn.IsMethod || fn.Name == "" {
				continue
			}
			fmt.Fprintf(&out, "\t%s(%s)%s\n", fn.Name, goParams(fn.Params), goReturns(fn.Returns))
		}
		out.WriteString("}\n")
		return out.String()
	}
	fmt.Fprintf(&out, "type %s struct{}\n", className)
	for _, fn := range anatomy.Functions {
		if fn.Name == "" || strings.HasPrefix(fn.Name, "New") {
			continue
		}
		out.WriteString("\n")
		if fn.IsMethod {
			fmt.Fprintf(&out, "func (x *%s) %s(%s)%s {\n", className, fn.Name, goParams(fn.Params), goReturns(fn.Returns))
		} else {
			fmt.Fprintf(&out, "func %s(%s)%s {\n", fn.Name, goParams(fn.Params), goReturns(fn.Returns))
		}
		if ret := goReturnStub(fn.Returns); ret != "" {
			fmt.Fprintf(&out, "\t%s\n", ret)
		}
		out.WriteString("}\n")
	}
	return out.String()
}

func nonFeatureImports(imports []string) []string {
	var out []string
	for _, imp := range imports {
		if strings.Contains(imp, "/features/") || strings.Contains(imp, "../") {
			continue
		}
		out = append(out, imp)
	}
	return out
}

func dartReturnType(fn models.FunctionSignature) string {
	if fn.ReturnType != "" {
		return fn.ReturnType
	}
	return "void"
}

func dartParams(params []models.Parameter) string {
	var parts []string
	for _, p := range params {
		switch {
		case p.Type != "" && p.Name != "":
			parts = append(parts, p.Type+" "+p.Name)
		case p.Name != "":
			parts = append(parts, p.Name)
		case p.Type != "":
			parts = append(parts, p.Type)
		}
	}
	return strings.Join(parts, ", ")
}

func dartReturnStub(returnType string) string {
	switch {
	case returnType == "" || returnType == "void":
		return ""
	case strings.HasPrefix(returnType, "Future<"):
		return "throw UnimplementedError();"
	case returnType == "bool":
		return "return false;"
	case returnType == "int":
		return "return 0;"
	case returnType == "double":
		return "return 0;"
	case returnType == "String":
		return "return '';"
	default:
		return "throw UnimplementedError();"
	}
}

func goImportsForFunctions(anatomy models.FileAnatomy) []string {
	needsContext := false
	for _, fn := range anatomy.Functions {
		for _, p := range fn.Params {
			if strings.Contains(p.Type, "context.Context") {
				needsContext = true
			}
		}
		if strings.Contains(fn.Returns, "context.Context") {
			needsContext = true
		}
	}
	if needsContext {
		return []string{"context"}
	}
	return nil
}

func goParams(params []models.Parameter) string {
	var parts []string
	for _, p := range params {
		switch {
		case p.Name != "" && p.Type != "":
			parts = append(parts, p.Name+" "+p.Type)
		case p.Type != "":
			parts = append(parts, p.Type)
		case p.Name != "":
			parts = append(parts, p.Name)
		}
	}
	return strings.Join(parts, ", ")
}

func goReturns(returns string) string {
	returns = strings.TrimSpace(returns)
	if returns == "" {
		return ""
	}
	if strings.HasPrefix(returns, "(") {
		return " " + returns
	}
	return " " + returns
}

func goReturnStub(returns string) string {
	returns = strings.TrimSpace(returns)
	if returns == "" {
		return ""
	}
	if strings.Contains(returns, "error") {
		return "return nil"
	}
	if strings.HasPrefix(returns, "*") || strings.HasPrefix(returns, "[]") || strings.HasPrefix(returns, "map[") {
		return "return nil"
	}
	switch returns {
	case "bool":
		return "return false"
	case "int", "int64", "float64":
		return "return 0"
	case "string":
		return `return ""`
	default:
		return "return nil"
	}
}

func startsWithUnderscore(s string) bool {
	return strings.HasPrefix(s, "_")
}
