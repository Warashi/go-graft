package projectload

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadIncludesTestFilesAndAbsolutePaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/m\n\ngo 1.26.0\n")
	writeFile(t, filepath.Join(dir, "pkg", "calc.go"), "package pkg\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, filepath.Join(dir, "pkg", "calc_test.go"), "package pkg\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) { _ = Add(1, 2) }\n")
	writeFile(t, filepath.Join(dir, "pkg", "calc_external_test.go"), "package pkg_test\n\nimport \"testing\"\n\nfunc TestExternal(t *testing.T) {}\n")

	loader := Loader{Dir: dir}
	project, err := loader.Load(context.Background(), "./pkg")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(project.Packages) == 0 {
		t.Fatal("Load() returned 0 packages")
	}

	var hasCalc, hasCalcTest, hasExternal bool
	for _, pkg := range project.Packages {
		for _, file := range pkg.GoFiles {
			if !filepath.IsAbs(file) {
				t.Fatalf("GoFiles path is not absolute: %s", file)
			}
			switch filepath.Base(file) {
			case "calc.go":
				hasCalc = true
			case "calc_test.go":
				hasCalcTest = true
			case "calc_external_test.go":
				hasExternal = true
			}
		}
		for _, file := range pkg.CompiledGoFiles {
			if !filepath.IsAbs(file) {
				t.Fatalf("CompiledGoFiles path is not absolute: %s", file)
			}
		}
	}

	if !hasCalc || !hasCalcTest || !hasExternal {
		t.Fatalf("missing files: calc=%v calc_test=%v external_test=%v", hasCalc, hasCalcTest, hasExternal)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}
