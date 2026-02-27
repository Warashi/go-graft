package astcow

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestClonePathCreatesCopyOnWriteFile(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package p
var top = 1 + 2
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	gen := file.Decls[0].(*ast.GenDecl)
	spec := gen.Specs[0].(*ast.ValueSpec)
	origExpr := spec.Values[0].(*ast.BinaryExpr)
	mutatedExpr := *origExpr
	mutatedExpr.Op = token.SUB

	path := []ast.Node{file, gen, spec, origExpr}
	clonedFile, cloneMap, err := ClonePath(path, origExpr, &mutatedExpr)
	if err != nil {
		t.Fatalf("ClonePath() error = %v", err)
	}

	if got := spec.Values[0].(*ast.BinaryExpr).Op; got != token.ADD {
		t.Fatalf("original op = %v, want +", got)
	}
	clonedExpr := clonedFile.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Values[0].(*ast.BinaryExpr)
	if clonedExpr.Op != token.SUB {
		t.Fatalf("cloned op = %v, want -", clonedExpr.Op)
	}
	if cloneMap[clonedFile] != file {
		t.Fatal("cloneMap should keep file origin mapping")
	}
}

func TestClonePathCopiesSliceFields(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package p
var top = 1 + 2
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	gen := file.Decls[0].(*ast.GenDecl)
	spec := gen.Specs[0].(*ast.ValueSpec)
	origExpr := spec.Values[0].(*ast.BinaryExpr)
	mutatedExpr := *origExpr
	mutatedExpr.Op = token.SUB

	path := []ast.Node{file, gen, spec, origExpr}
	clonedFile, _, err := ClonePath(path, origExpr, &mutatedExpr)
	if err != nil {
		t.Fatalf("ClonePath() error = %v", err)
	}

	clonedFile.Decls[0] = nil
	if file.Decls[0] == nil {
		t.Fatal("original file.Decls should stay unchanged")
	}
}
