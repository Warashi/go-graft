package testselect

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutationpoint"
	"github.com/Warashi/go-graft/internal/projectload"
	"github.com/Warashi/go-graft/internal/testdiscover"
)

func TestNewSelectorWithOptionsFallsBackToASTWhenCHAFails(t *testing.T) {
	project := &model.Project{
		Packages: []*model.Package{
			{ID: "p", ImportPath: "example.com/p"},
		},
	}
	tests := []model.TestRef{
		{PkgID: "p", ImportPath: "example.com/p", Name: "TestP"},
	}

	selector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeCHA,
	})
	if selector.ResolvedBackend() != "ast" {
		t.Fatalf("resolved backend = %q, want ast", selector.ResolvedBackend())
	}
	if len(selector.BuildFailures()) == 0 {
		t.Fatal("BuildFailures() should not be empty")
	}
}

func TestSelectWithCHAFindsInterfaceDispatch(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/p.go", `package p
type worker interface {
	Do() int
}
type impl struct{}
func (impl) Do() int { return 1 }
func target() int {
	var w worker = impl{}
	return w.Do()
}
func Touch() int { return target() }
`)
	writeModuleFile(t, moduleDir, "p/p_test.go", `package p
import "testing"
func TestReachable(t *testing.T) {
	if Touch() != 1 {
		t.Fatal("bad")
	}
}
func TestUnrelated(t *testing.T) {}
`)

	project, err := (projectload.Loader{Dir: moduleDir}).Load(context.Background(), "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	tests := testdiscover.Discover(project)
	points := mutationpoint.Collect(project, []reflect.Type{reflect.TypeOf(&ast.BasicLit{})})
	point, ok := findPointInFunc(points, "Do")
	if !ok {
		t.Fatal("mutation point in Do not found")
	}

	astSelector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeAST,
	})
	chaSelector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeCHA,
	})
	if chaSelector.ResolvedBackend() != "cha" {
		t.Fatalf("resolved backend = %q, want cha", chaSelector.ResolvedBackend())
	}

	astNames := selectedNames(astSelector.Select(point), "example.com/m/p")
	chaNames := selectedNames(chaSelector.Select(point), "example.com/m/p")

	if !slices.Equal(astNames, []string{"TestReachable", "TestUnrelated"}) {
		t.Fatalf("ast selected = %v, want [TestReachable TestUnrelated]", astNames)
	}
	if !slices.Equal(chaNames, []string{"TestReachable"}) {
		t.Fatalf("cha selected = %v, want [TestReachable]", chaNames)
	}
}

func findPointInFunc(points []model.MutationPoint, funcName string) (model.MutationPoint, bool) {
	for _, point := range points {
		if point.EnclosingFunc == nil || point.EnclosingFunc.Name == nil {
			continue
		}
		if point.EnclosingFunc.Name.Name != funcName {
			continue
		}
		return point, true
	}
	return model.MutationPoint{}, false
}

func selectedNames(selected model.SelectedTests, importPath string) []string {
	names := append([]string(nil), selected.ByImportPath[importPath]...)
	slices.Sort(names)
	return names
}

func writeModuleFile(t *testing.T, moduleDir string, rel string, content string) {
	t.Helper()
	path := filepath.Join(moduleDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

