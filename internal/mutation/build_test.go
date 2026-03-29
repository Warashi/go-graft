package mutation

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestBuilderBuildWritesOverlayAndMutantFile(t *testing.T) {
	fset := token.NewFileSet()
	origPath := filepath.Join(t.TempDir(), "orig.go")
	file, err := parser.ParseFile(fset, origPath, `package p
var x = 1 + 2
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	binary := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.BinaryExpr)
	binary.Op = token.SUB

	builder := Builder{BaseTempDir: t.TempDir()}
	point := model.MutationPoint{
		FilePath:       origPath,
		CompiledGoFile: origPath,
	}
	mutant, err := builder.Build(Input{
		ID:       "m-1",
		RuleName: "bin-op",
		Point:    point,
		Mutated:  file,
		Fset:     fset,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if _, err := os.Stat(mutant.MutantFile); err != nil {
		t.Fatalf("mutant file stat error: %v", err)
	}
	if _, err := os.Stat(mutant.OverlayTmpDir); err != nil {
		t.Fatalf("tmp dir stat error: %v", err)
	}

	raw, err := os.ReadFile(mutant.OverlayPath)
	if err != nil {
		t.Fatalf("ReadFile(overlay) error = %v", err)
	}
	var payload struct {
		Replace map[string]string `json:"Replace"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal(overlay) error = %v", err)
	}
	got, ok := payload.Replace[origPath]
	if !ok {
		t.Fatalf("overlay key %q missing", origPath)
	}
	if got != mutant.MutantFile {
		t.Fatalf("overlay value = %q, want %q", got, mutant.MutantFile)
	}
}
