package graft

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestContextOriginalResolvesTransitiveCloneChain(t *testing.T) {
	original := &ast.Ident{Name: "orig"}
	clone1 := &ast.Ident{Name: "clone1"}
	clone2 := &ast.Ident{Name: "clone2"}

	ctx := newContext()
	ctx.setOriginal(clone1, original)
	ctx.setOriginal(clone2, clone1)

	got := ctx.Original(clone2)
	if got != original {
		t.Fatalf("Original(clone2) = %T %p, want %T %p", got, got, original, original)
	}
}

func TestContextTypeOfResolvesTransitiveCloneChain(t *testing.T) {
	original, info := parseTypedBinaryExpr(t, `package p
func f(a int, b int) int { return a + b }
`)

	clone1 := *original
	clone2 := clone1

	ctx := newContext()
	ctx.Types = info
	ctx.setOriginal(&clone1, original)
	ctx.setOriginal(&clone2, &clone1)

	got := ctx.TypeOf(&clone2)
	if got == nil {
		t.Fatal("TypeOf(clone2) = nil, want int")
	}
	if got.String() != "int" {
		t.Fatalf("TypeOf(clone2) = %q, want %q", got.String(), "int")
	}
}

func TestContextOriginalStopsOnCycle(t *testing.T) {
	clone1 := &ast.Ident{Name: "clone1"}
	clone2 := &ast.Ident{Name: "clone2"}

	ctx := newContext()
	ctx.setOriginal(clone1, clone2)
	ctx.setOriginal(clone2, clone1)

	got := ctx.Original(clone1)
	if got != clone1 {
		t.Fatalf("Original(clone1) = %T %p, want %T %p", got, got, clone1, clone1)
	}
}

func parseTypedBinaryExpr(t *testing.T, src string) (*ast.BinaryExpr, *types.Info) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	conf := types.Config{}
	if _, err := conf.Check("example.com/p", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("decl type = %T, want *ast.FuncDecl", file.Decls[0])
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("stmt type = %T, want *ast.ReturnStmt", fn.Body.List[0])
	}
	expr, ok := ret.Results[0].(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expr type = %T, want *ast.BinaryExpr", ret.Results[0])
	}
	return expr, info
}
