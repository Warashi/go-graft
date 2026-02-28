package testselect

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestSelectPrunesByReverseDependencies(t *testing.T) {
	project := &model.Project{
		Packages: []*model.Package{
			{ID: "a", ImportPath: "example.com/a", Imports: nil},
			{ID: "b", ImportPath: "example.com/b", Imports: []string{"example.com/a"}},
			{ID: "c", ImportPath: "example.com/c", Imports: nil},
		},
	}
	tests := []model.TestRef{
		{PkgID: "a", ImportPath: "example.com/a", Name: "TestA"},
		{PkgID: "b", ImportPath: "example.com/b", Name: "TestB"},
		{PkgID: "c", ImportPath: "example.com/c", Name: "TestC"},
	}
	point := model.MutationPoint{PkgID: "a", PkgImportPath: "example.com/a"}

	selected := Select(project, tests, point)
	if got := selected.ByImportPath["example.com/a"]; !reflect.DeepEqual(got, []string{"TestA"}) {
		t.Fatalf("pkg a tests = %v, want [TestA]", got)
	}
	if got := selected.ByImportPath["example.com/b"]; !reflect.DeepEqual(got, []string{"TestB"}) {
		t.Fatalf("pkg b tests = %v, want [TestB]", got)
	}
	if _, ok := selected.ByImportPath["example.com/c"]; ok {
		t.Fatal("pkg c should be pruned")
	}
}

func TestSelectUsesReverseCallersFromSeedFunction(t *testing.T) {
	const src = `package a
import "testing"
func target() {}
func helper() { target() }
func TestTarget(t *testing.T) { helper() }
func TestUnrelated(t *testing.T) {}
`
	filePath := filepath.Clean("/tmp/a_test.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	project := &model.Project{
		Packages: []*model.Package{
			{
				ID:           "a",
				ImportPath:   "example.com/a",
				GoFiles:      []string{filePath},
				SyntaxByPath: map[string]*ast.File{filePath: file},
				Imports:      nil,
			},
		},
	}
	tests := []model.TestRef{
		{PkgID: "a", ImportPath: "example.com/a", Name: "TestTarget"},
		{PkgID: "a", ImportPath: "example.com/a", Name: "TestUnrelated"},
	}
	point := model.MutationPoint{
		PkgID:         "a",
		PkgImportPath: "example.com/a",
		EnclosingFunc: &ast.FuncDecl{Name: ast.NewIdent("target")},
	}

	selected := Select(project, tests, point)
	got := selected.ByImportPath["example.com/a"]
	if !reflect.DeepEqual(got, []string{"TestTarget"}) {
		t.Fatalf("selected tests = %v, want [TestTarget]", got)
	}
}

func TestSelectReturnsEmptyWhenNoDependentTests(t *testing.T) {
	project := &model.Project{
		Packages: []*model.Package{
			{ID: "x", ImportPath: "example.com/x"},
		},
	}
	tests := []model.TestRef{
		{PkgID: "y", ImportPath: "example.com/y", Name: "TestY"},
	}
	point := model.MutationPoint{PkgImportPath: "example.com/x"}

	selected := Select(project, tests, point)
	if len(selected.ByImportPath) != 0 {
		t.Fatalf("selected map should be empty, got %v", selected.ByImportPath)
	}
}

func TestResolveCallPrefersCurrentPackageIDWhenImportPathHasMultipleCandidates(t *testing.T) {
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

	got, ok := resolveCall("example.com/dep", call, aliases, byImport)
	if !ok {
		t.Fatal("resolveCall() should resolve selector call")
	}
	want := functionKey{pkgID: "example.com/dep", name: "Target"}
	if got != want {
		t.Fatalf("resolveCall() = %+v, want %+v", got, want)
	}
}
