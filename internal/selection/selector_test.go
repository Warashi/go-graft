package selection

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutation"
	"github.com/Warashi/go-graft/internal/project"
)

func TestNewSelectorWithOptionsFallsBackToASTWhenRTAAndCHAFail(t *testing.T) {
	project := &project.Project{
		Packages: []*project.Package{
			{ID: "p", ImportPath: "example.com/p"},
		},
	}
	tests := []model.TestRef{
		{PkgID: "p", ImportPath: "example.com/p", Name: "TestP"},
	}

	selector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeRTA,
	})
	if selector.ResolvedBackend() != "ast" {
		t.Fatalf("resolved backend = %q, want ast", selector.ResolvedBackend())
	}
	failures := selector.BuildFailures()
	if len(failures) != 2 {
		t.Fatalf("BuildFailures() len = %d, want 2 (%v)", len(failures), failures)
	}
	if !strings.HasPrefix(failures[0], "rta:") {
		t.Fatalf("first failure = %q, want rta:*", failures[0])
	}
	if !strings.HasPrefix(failures[1], "cha:") {
		t.Fatalf("second failure = %q, want cha:*", failures[1])
	}
}

func TestNewSelectorWithOptionsResolvesBackendByMode(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/p.go", `package p
func Entry() int {
	return helper()
}
func helper() int { return 1 }
`)

	project, err := (project.Loader{Dir: moduleDir}).Load(context.Background(), "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	pkg := findPackageByImportPath(project, "example.com/m/p")
	if pkg == nil {
		t.Fatal("package example.com/m/p not found")
	}
	tests := []model.TestRef{
		{PkgID: pkg.ID, ImportPath: pkg.ImportPath, Name: "Entry"},
	}

	astSelector := NewSelectorWithOptions(project, tests, SelectorOptions{CallGraphMode: CallGraphModeAST})
	chaSelector := NewSelectorWithOptions(project, tests, SelectorOptions{CallGraphMode: CallGraphModeCHA})
	rtaSelector := NewSelectorWithOptions(project, tests, SelectorOptions{CallGraphMode: CallGraphModeRTA})
	autoSelector := NewSelectorWithOptions(project, tests, SelectorOptions{CallGraphMode: CallGraphModeAuto})

	if astSelector.ResolvedBackend() != "ast" {
		t.Fatalf("ast selector resolved backend = %q, want ast", astSelector.ResolvedBackend())
	}
	if chaSelector.ResolvedBackend() != "cha" {
		t.Fatalf("cha selector resolved backend = %q, want cha", chaSelector.ResolvedBackend())
	}
	if rtaSelector.ResolvedBackend() != "rta" {
		t.Fatalf("rta selector resolved backend = %q, want rta", rtaSelector.ResolvedBackend())
	}
	if autoSelector.ResolvedBackend() != "rta" {
		t.Fatalf("auto selector resolved backend = %q, want rta", autoSelector.ResolvedBackend())
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

	project, err := (project.Loader{Dir: moduleDir}).Load(context.Background(), "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	tests := Discover(project)
	points := mutation.Collect(project, []reflect.Type{reflect.TypeFor[*ast.BasicLit]()})
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

func TestSelectWithRTAFindsFunctionValueCall(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/p.go", `package p
func callee() int { return 1 }
func callViaFunc() int {
	f := callee
	return f()
}
func Touch() int { return callViaFunc() }
func EntryReachable() int { return Touch() }
func EntryUnrelated() int { return 0 }
`)

	project, err := (project.Loader{Dir: moduleDir}).Load(context.Background(), "./...")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	pkg := findPackageByImportPath(project, "example.com/m/p")
	if pkg == nil {
		t.Fatal("package example.com/m/p not found")
	}
	tests := []model.TestRef{
		{PkgID: pkg.ID, ImportPath: pkg.ImportPath, Name: "EntryReachable"},
		{PkgID: pkg.ID, ImportPath: pkg.ImportPath, Name: "EntryUnrelated"},
	}
	points := mutation.Collect(project, []reflect.Type{reflect.TypeFor[*ast.BasicLit]()})
	point, ok := findPointInFunc(points, "callee")
	if !ok {
		t.Fatal("mutation point in callee not found")
	}

	astSelector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeAST,
	})
	rtaSelector := NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeRTA,
	})
	if rtaSelector.ResolvedBackend() != "rta" {
		t.Fatalf("resolved backend = %q, want rta", rtaSelector.ResolvedBackend())
	}

	astNames := selectedNames(astSelector.Select(point), "example.com/m/p")
	rtaNames := selectedNames(rtaSelector.Select(point), "example.com/m/p")

	if !slices.Equal(astNames, []string{"EntryReachable", "EntryUnrelated"}) {
		t.Fatalf("ast selected = %v, want [EntryReachable EntryUnrelated]", astNames)
	}
	if !slices.Equal(rtaNames, []string{"EntryReachable"}) {
		t.Fatalf("rta selected = %v, want [EntryReachable]", rtaNames)
	}
}

func findPackageByImportPath(project *project.Project, importPath string) *project.Package {
	if project == nil {
		return nil
	}
	return project.ByImportPath[importPath]
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
