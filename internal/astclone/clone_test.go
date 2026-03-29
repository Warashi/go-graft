package astclone

import (
	"errors"
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

func TestDeepCopyNodeCopiesSubtreeAndTracksOriginals(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", `package p
var top = (1 + 2) + (3 + 4)
`, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	gen := file.Decls[0].(*ast.GenDecl)
	spec := gen.Specs[0].(*ast.ValueSpec)
	origRoot := spec.Values[0].(*ast.BinaryExpr)
	origLeftParen := origRoot.X.(*ast.ParenExpr)
	origLeft := origLeftParen.X.(*ast.BinaryExpr)

	clonedNode, cloneMap, err := DeepCopyNode(origRoot)
	if err != nil {
		t.Fatalf("DeepCopyNode() error = %v", err)
	}
	clonedRoot, ok := clonedNode.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("cloned root type = %T, want *ast.BinaryExpr", clonedNode)
	}
	clonedLeftParen, ok := clonedRoot.X.(*ast.ParenExpr)
	if !ok {
		t.Fatalf("cloned left paren type = %T, want *ast.ParenExpr", clonedRoot.X)
	}
	clonedLeft, ok := clonedLeftParen.X.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("cloned left type = %T, want *ast.BinaryExpr", clonedLeftParen.X)
	}

	if cloneMap[clonedRoot] != origRoot {
		t.Fatal("cloneMap should map cloned root back to original root")
	}
	if cloneMap[clonedLeft] != origLeft {
		t.Fatal("cloneMap should map cloned child back to original child")
	}

	clonedLeft.Op = token.SUB
	if origLeft.Op != token.ADD {
		t.Fatalf("original child op = %v, want +", origLeft.Op)
	}
}

func TestDeepCopyNodePreservesSharedReferences(t *testing.T) {
	shared := &ast.Ident{Name: "x"}
	orig := &ast.BinaryExpr{
		X:  shared,
		Op: token.ADD,
		Y:  shared,
	}

	clonedNode, cloneMap, err := DeepCopyNode(orig)
	if err != nil {
		t.Fatalf("DeepCopyNode() error = %v", err)
	}
	cloned, ok := clonedNode.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("cloned type = %T, want *ast.BinaryExpr", clonedNode)
	}
	clonedX, ok := cloned.X.(*ast.Ident)
	if !ok {
		t.Fatalf("cloned X type = %T, want *ast.Ident", cloned.X)
	}
	clonedY, ok := cloned.Y.(*ast.Ident)
	if !ok {
		t.Fatalf("cloned Y type = %T, want *ast.Ident", cloned.Y)
	}

	if clonedX != clonedY {
		t.Fatal("shared reference should stay shared in cloned tree")
	}
	if cloneMap[clonedX] != shared {
		t.Fatal("cloneMap should map shared clone to original shared node")
	}
	if cloneMap[cloned] != orig {
		t.Fatal("cloneMap should map root clone to original root")
	}
}

func TestDeepCopyNodeErrorsOnUnsupportedNode(t *testing.T) {
	_, _, err := DeepCopyNode(&unsupportedNode{})
	if err == nil {
		t.Fatal("DeepCopyNode() error = nil, want error")
	}
	if !errors.Is(err, ErrUnsupportedNode) {
		t.Fatalf("DeepCopyNode() error = %v, want ErrUnsupportedNode", err)
	}
}

type unsupportedNode struct{}

func (*unsupportedNode) Pos() token.Pos { return 0 }
func (*unsupportedNode) End() token.Pos { return 0 }
