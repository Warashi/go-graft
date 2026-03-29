package execution

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/go-graft/internal/model"
)

func TestBuildRunPatternQuotesAndSorts(t *testing.T) {
	got := buildRunPattern([]string{"TestB", "TestA", "TestA/Sub"})
	want := "^(TestA|TestA/Sub|TestB)$"
	if got != want {
		t.Fatalf("buildRunPattern() = %q, want %q", got, want)
	}
}

func TestRunMarksUnsupportedForEmptySelection(t *testing.T) {
	r := Runner{Workers: 1}
	results := r.Run(context.Background(), []model.Mutant{{ID: "m1"}})
	if len(results) != 1 {
		t.Fatalf("Run() results len = %d, want 1", len(results))
	}
	if results[0].Status != model.MutantUnsupported {
		t.Fatalf("status = %v, want %v", results[0].Status, model.MutantUnsupported)
	}
}

func TestRunKillsMutantOnFailingTest(t *testing.T) {
	moduleDir := t.TempDir()
	writeFile(t, filepath.Join(moduleDir, "go.mod"), "module example.com/m\n\ngo 1.26.0\n")
	sourcePath := filepath.Join(moduleDir, "p", "add.go")
	writeFile(t, sourcePath, "package p\n\nfunc Add(a, b int) int { return a + b }\n")
	writeFile(t, filepath.Join(moduleDir, "p", "add_test.go"), "package p\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 1) != 3 {\n\t\tt.Fatal(\"bad\")\n\t}\n}\n")

	mutant := newMutantFixture(t, moduleDir, sourcePath, "package p\n\nfunc Add(a, b int) int { return a - b }\n")
	mutant.SelectedTests = model.SelectedTests{ByImportPath: map[string][]string{
		"./p": {"TestAdd"},
	}}

	r := Runner{
		Workers:       1,
		MutantTimeout: 10 * time.Second,
		KeepTemp:      true,
	}
	results := r.Run(context.Background(), []model.Mutant{mutant})
	got := results[0]
	if got.Status != model.MutantKilled {
		t.Fatalf("status = %v, want %v", got.Status, model.MutantKilled)
	}
	if got.TimedOut {
		t.Fatal("TimedOut should be false")
	}
	assertCommandContains(t, got.FailedCommand, "-parallel=1")
	assertCommandContains(t, got.FailedCommand, "-failfast")
	assertCommandContains(t, got.FailedCommand, "-overlay="+mutant.OverlayPath)
}

func TestRunMarksTimedOutAsKilled(t *testing.T) {
	moduleDir := t.TempDir()
	writeFile(t, filepath.Join(moduleDir, "go.mod"), "module example.com/m\n\ngo 1.26.0\n")
	sourcePath := filepath.Join(moduleDir, "p", "hang.go")
	writeFile(t, sourcePath, "package p\n\nfunc Busy() {}\n")
	writeFile(t, filepath.Join(moduleDir, "p", "hang_test.go"), "package p\n\nimport \"testing\"\n\nfunc TestHang(t *testing.T) {\n\tselect {}\n}\n")

	mutant := newMutantFixture(t, moduleDir, sourcePath, "package p\n\nfunc Busy() {}\n")
	mutant.SelectedTests = model.SelectedTests{ByImportPath: map[string][]string{
		"./p": {"TestHang"},
	}}

	r := Runner{
		Workers:       1,
		MutantTimeout: 300 * time.Millisecond,
		KeepTemp:      true,
	}
	results := r.Run(context.Background(), []model.Mutant{mutant})
	got := results[0]
	if got.Status != model.MutantKilled {
		t.Fatalf("status = %v, want %v", got.Status, model.MutantKilled)
	}
	if !got.TimedOut {
		t.Fatal("TimedOut should be true")
	}
}

func newMutantFixture(t *testing.T, moduleDir string, originalPath string, mutatedContent string) model.Mutant {
	t.Helper()
	tempDir := t.TempDir()
	overlayDir := filepath.Join(tempDir, "overlay")
	tmpDir := filepath.Join(tempDir, "tmp")
	if err := os.MkdirAll(overlayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(overlay) error = %v", err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(tmp) error = %v", err)
	}

	mutantPath := filepath.Join(overlayDir, filepath.Base(originalPath))
	writeFile(t, mutantPath, mutatedContent)
	overlayPath := filepath.Join(tempDir, "overlay.json")
	writeFile(t, overlayPath, "{\n  \"Replace\": {\n    \""+originalPath+"\": \""+mutantPath+"\"\n  }\n}\n")

	return model.Mutant{
		ID:            "m-fixture",
		WorkDir:       moduleDir,
		TempDir:       tempDir,
		OverlayPath:   overlayPath,
		OverlayTmpDir: tmpDir,
		MutantFile:    mutantPath,
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func assertCommandContains(t *testing.T, command []string, want string) {
	t.Helper()
	if slices.Contains(command, want) {
		return
	}
	t.Fatalf("command %v does not contain %q", strings.Join(command, " "), want)
}
