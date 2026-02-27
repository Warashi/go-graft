package testdiscover

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Warashi/go-graft/internal/model"
)

func Discover(project *model.Project) []model.TestRef {
	if project == nil {
		return nil
	}
	out := make([]model.TestRef, 0)
	for _, pkg := range project.Packages {
		for _, filePath := range pkg.GoFiles {
			file := pkg.SyntaxByPath[filePath]
			if file == nil {
				continue
			}
			testingAliases := testingImportAliases(file)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if !isTopLevelTest(fn, testingAliases, pkg.TypesInfo) {
					continue
				}
				out = append(out, model.TestRef{
					PkgID:       pkg.ID,
					ImportPath:  pkg.ImportPath,
					Name:        fn.Name.Name,
					FilePath:    filepath.Clean(filePath),
					PackageName: file.Name.Name,
				})
			}
		}
	}
	return out
}

func isTopLevelTest(fn *ast.FuncDecl, testingAliases map[string]struct{}, typesInfo *types.Info) bool {
	if fn == nil || fn.Name == nil || fn.Recv != nil || fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	if !looksLikeTestName(fn.Name.Name) {
		return false
	}
	if len(fn.Type.Params.List) != 1 {
		return false
	}
	if len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}
	return isPointerToTestingT(fn.Type.Params.List[0].Type, testingAliases, typesInfo)
}

func looksLikeTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	rest := strings.TrimPrefix(name, "Test")
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(r)
}

func isPointerToTestingT(expr ast.Expr, testingAliases map[string]struct{}, typesInfo *types.Info) bool {
	if isPointerToTestingTByTypes(expr, typesInfo) {
		return true
	}

	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if sel.Sel == nil || sel.Sel.Name != "T" {
		return false
	}
	_, ok = testingAliases[pkgIdent.Name]
	return ok
}

func isPointerToTestingTByTypes(expr ast.Expr, info *types.Info) bool {
	if info == nil || expr == nil {
		return false
	}
	typ := types.Unalias(info.TypeOf(expr))
	ptr, ok := typ.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "testing" && obj.Name() == "T"
}

func testingImportAliases(file *ast.File) map[string]struct{} {
	out := map[string]struct{}{
		"testing": {},
	}
	if file == nil {
		return out
	}
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, "\"")
		if path != "testing" {
			continue
		}
		if imp.Name == nil {
			out["testing"] = struct{}{}
			continue
		}
		if imp.Name.Name == "." || imp.Name.Name == "_" {
			continue
		}
		out[imp.Name.Name] = struct{}{}
	}
	return out
}
