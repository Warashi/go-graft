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
