package callresolve

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestImportAliases(t *testing.T) {
	file := &ast.File{
		Imports: []*ast.ImportSpec{
			{Path: &ast.BasicLit{Value: "\"example.com/foo\""}},
			{Name: ast.NewIdent("bar"), Path: &ast.BasicLit{Value: "\"example.com/bar\""}},
			{Name: ast.NewIdent("_"), Path: &ast.BasicLit{Value: "\"example.com/ignored\""}},
			{Name: ast.NewIdent("."), Path: &ast.BasicLit{Value: "\"example.com/dot\""}},
		},
	}
	got := ImportAliases(file)
	want := map[string]string{
		"foo": "example.com/foo",
		"bar": "example.com/bar",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ImportAliases() = %v, want %v", got, want)
	}
}

func TestBuildByImportSortsAndDeduplicates(t *testing.T) {
	project := &model.Project{
		Packages: []*model.Package{
			{ID: "b", ImportPath: "example.com/p"},
			{ID: "a", ImportPath: "example.com/p"},
			{ID: "a", ImportPath: "example.com/p"},
			nil,
		},
	}
	got := BuildByImport(project)
	want := map[string][]string{
		"example.com/p": {"a", "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildByImport() = %v, want %v", got, want)
	}
}

func TestResolveFunctionCallPrefersCurrentPackageID(t *testing.T) {
	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("dep"),
			Sel: ast.NewIdent("Target"),
		},
	}
	aliases := map[string]string{
		"dep": "example.com/dep",
	}
	byImport := map[string][]string{
		"example.com/dep": {"example.com/dep.test", "example.com/dep"},
	}

	got, ok := ResolveFunctionCall("example.com/dep", call, nil, aliases, byImport, nil)
	if !ok {
		t.Fatal("ResolveFunctionCall() should resolve selector call")
	}
	want := FunctionKey{PkgID: "example.com/dep", Name: "Target"}
	if got != want {
		t.Fatalf("ResolveFunctionCall() = %+v, want %+v", got, want)
	}
}

func TestResolveFunctionCallFiltersByExists(t *testing.T) {
	call := &ast.CallExpr{
		Fun: ast.NewIdent("helper"),
	}
	byImport := map[string][]string{
		"example.com/current": {"example.com/current"},
	}
	exists := func(key FunctionKey) bool {
		return key.PkgID == "example.com/current" && key.Name == "helper2"
	}

	if _, ok := ResolveFunctionCall("example.com/current", call, nil, nil, byImport, exists); ok {
		t.Fatal("ResolveFunctionCall() should return false for missing key")
	}
}

func TestResolveFunctionCallUsesTypesBeforeSyntax(t *testing.T) {
	sel := &ast.SelectorExpr{
		X:   ast.NewIdent("wrongAlias"),
		Sel: ast.NewIdent("Target"),
	}
	call := &ast.CallExpr{Fun: sel}
	typesPkg := types.NewPackage("example.com/dep", "dep")
	typesFn := types.NewFunc(token.NoPos, typesPkg, "Target", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	info := &types.Info{
		Uses: map[*ast.Ident]types.Object{
			sel.Sel: typesFn,
		},
	}
	aliases := map[string]string{
		"wrongAlias": "example.com/other",
	}
	byImport := map[string][]string{
		"example.com/dep": {"dep.test", "dep"},
	}

	got, ok := ResolveFunctionCall("dep", call, info, aliases, byImport, nil)
	if !ok {
		t.Fatal("ResolveFunctionCall() should resolve call by types info")
	}
	want := FunctionKey{PkgID: "dep", Name: "Target"}
	if got != want {
		t.Fatalf("ResolveFunctionCall() = %+v, want %+v", got, want)
	}
}
