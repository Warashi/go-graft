package graft

import (
	"context"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEngineRunKilled(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/add.go", "package p\n\nfunc Add(a, b int) int { return a + b }\n")
	writeModuleFile(t, moduleDir, "p/add_test.go", "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, WithName("add-to-sub"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Killed == 0 {
		t.Fatalf("killed = %d, want > 0 (report=%+v)", report.Killed, report)
	}
}

func TestEngineRunSurvived(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/add.go", "package p\n\nfunc Add(a, b int) int { return a + b }\n")
	writeModuleFile(t, moduleDir, "p/add_test.go", "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.ADD
		return n, true
	}, WithName("add-to-add"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Survived == 0 {
		t.Fatalf("survived = %d, want > 0 (report=%+v)", report.Survived, report)
	}
}

func TestEngineRunUnsupportedWhenNoDependentTests(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "a/add.go", "package a\n\nfunc Add(a, b int) int { return a + b }\n")
	writeModuleFile(t, moduleDir, "b/b_test.go", "package b\n\nimport \"testing\"\n\nfunc TestOnlyB(t *testing.T) {}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, WithName("add-to-sub"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Unsupported == 0 {
		t.Fatalf("unsupported = %d, want > 0 (report=%+v)", report.Unsupported, report)
	}
}

func TestEngineRunProvidesPackagesPackageInContext(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/add.go", "package p\n\nfunc Add(a, b int) int { return a + b }\n")
	writeModuleFile(t, moduleDir, "p/add_test.go", "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})

	var sawPkg bool
	Register[*ast.BinaryExpr](e, func(c *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		if c != nil && c.Pkg != nil && c.Pkg.TypesInfo != nil {
			sawPkg = true
		}
		n.Op = token.SUB
		return n, true
	}, WithName("ctx-pkg-check"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Total == 0 {
		t.Fatalf("total = %d, want > 0", report.Total)
	}
	if !sawPkg {
		t.Fatal("expected ctx.Pkg and ctx.Pkg.TypesInfo to be available in callback")
	}
}

func TestEngineRunTypeOfWorksOnCallbackNode(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/add.go", "package p\n\nfunc Add(a, b int) int { return a + b }\n")
	writeModuleFile(t, moduleDir, "p/add_test.go", "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 2 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})

	var sawTypeOnCallbackNode bool
	Register[*ast.BinaryExpr](e, func(c *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		typ := c.TypeOf(n)
		if typ == nil {
			return nil, false
		}
		if typ.String() != "int" {
			return nil, false
		}
		sawTypeOnCallbackNode = true
		return &ast.BinaryExpr{
			X:  n.X,
			Op: token.SUB,
			Y:  n.Y,
		}, true
	}, WithName("typed-add-to-sub-replace"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Total == 0 {
		t.Fatalf("total = %d, want > 0 (report=%+v)", report.Total, report)
	}
	if report.Killed == 0 {
		t.Fatalf("killed = %d, want > 0 (report=%+v)", report.Killed, report)
	}
	if report.Errored != 0 {
		t.Fatalf("errored = %d, want 0 (report=%+v)", report.Errored, report)
	}
	if !sawTypeOnCallbackNode {
		t.Fatal("expected ctx.TypeOf(n) to resolve callback node type")
	}
}

func TestEngineRunSkipsMutationsInTestFiles(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/add.go", "package p\n\nfunc Add() int { return 1 }\n")
	writeModuleFile(t, moduleDir, "p/add_test.go", "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add()+1 != 2 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	e := New(Config{
		Workers:       1,
		MutantTimeout: 5 * time.Second,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, WithName("add-to-sub"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Total != 0 {
		t.Fatalf("total = %d, want 0 (report=%+v)", report.Total, report)
	}
}

func TestEngineRunCallGraphModeChangesSelectedTests(t *testing.T) {
	moduleDir := t.TempDir()
	writeModuleFile(t, moduleDir, "go.mod", "module example.com/m\n\ngo 1.26.0\n")
	writeModuleFile(t, moduleDir, "p/p.go", `package p
type worker interface {
	Do() int
}
type impl struct{}
func (impl) Do() int { return 1 + 0 }
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

	astReport := runEngineWithCallGraphMode(t, moduleDir, TestSelectionCallGraphAST)
	chaReport := runEngineWithCallGraphMode(t, moduleDir, TestSelectionCallGraphCHA)
	autoReport := runEngineWithCallGraphMode(t, moduleDir, TestSelectionCallGraphAuto)

	const pkg = "example.com/m/p"
	astPattern := findRunPattern(astReport, pkg)
	chaPattern := findRunPattern(chaReport, pkg)
	autoPattern := findRunPattern(autoReport, pkg)

	if astPattern != "^(TestReachable|TestUnrelated)$" {
		t.Fatalf("ast run pattern = %q, want ^(TestReachable|TestUnrelated)$", astPattern)
	}
	if chaPattern != "^(TestReachable)$" {
		t.Fatalf("cha run pattern = %q, want ^(TestReachable)$", chaPattern)
	}
	if autoPattern != chaPattern {
		t.Fatalf("auto run pattern = %q, want %q", autoPattern, chaPattern)
	}
}

func runEngineWithCallGraphMode(t *testing.T, moduleDir string, mode TestSelectionCallGraphMode) *Report {
	t.Helper()
	e := New(Config{
		Workers:                1,
		MutantTimeout:          5 * time.Second,
		TestSelectionCallGraph: mode,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, WithName("add-to-sub"))

	report := runInDir(t, moduleDir, func() (*Report, error) {
		return e.Run(context.Background(), "./...")
	})
	if report.Total == 0 {
		t.Fatalf("total = %d, want > 0 (report=%+v)", report.Total, report)
	}
	return report
}

func findRunPattern(report *Report, pkg string) string {
	if report == nil {
		return ""
	}
	for _, mutant := range report.Mutants {
		for _, executed := range mutant.Executed {
			if executed.ImportPath != pkg {
				continue
			}
			return executed.RunPattern
		}
	}
	return ""
}

func runInDir(t *testing.T, dir string, fn func() (*Report, error)) *Report {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%s) error = %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore Chdir(%s) error = %v", prev, err)
		}
	}()

	report, err := fn()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return report
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
