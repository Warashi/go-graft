package graft

import (
	"context"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"testing"
	"time"
)

func TestEngineRunDogfoodRepository(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, WithName("builtin-add-to-sub"))

	report, err := e.Run(context.Background(), "./...")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Total == 0 {
		t.Fatalf("total = %d, want > 0", report.Total)
	}
	if report.Survived != 0 {
		t.Fatalf("survived = %d, want 0 (report=%+v)", report.Survived, report)
	}
	if report.Errored != 0 {
		t.Fatalf("errored = %d, want 0 (report=%+v)", report.Errored, report)
	}

	totalByStatus := 0
	for _, count := range []int{report.Killed, report.Survived, report.Unsupported, report.Errored} {
		totalByStatus += count
	}
	if totalByStatus != report.Total {
		t.Fatalf("status sum = %d, total = %d", totalByStatus, report.Total)
	}

	for _, mutant := range report.Mutants {
		for _, executed := range mutant.Executed {
			if strings.Contains(executed.RunPattern, "TestEngineRunDogfoodRepository") {
				t.Fatalf("run pattern should exclude dogfood test, got %q", executed.RunPattern)
			}
		}
	}
}

func TestEngineRunDogfoodMethodCallSwapMathBig(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	Register[*ast.CallExpr](e, func(c *Context, n *ast.CallExpr) (*ast.CallExpr, bool) {
		if c == nil || c.Types == nil {
			return nil, false
		}

		sel, ok := n.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Add" {
			return nil, false
		}

		originalCall, ok := c.Original(n).(*ast.CallExpr)
		if !ok {
			return nil, false
		}
		originalSel, ok := originalCall.Fun.(*ast.SelectorExpr)
		if !ok || originalSel.Sel == nil || originalSel.Sel.Name != "Add" {
			return nil, false
		}

		addFunc, ok := c.Types.ObjectOf(originalSel.Sel).(*types.Func)
		if !ok || addFunc == nil {
			return nil, false
		}
		addSig, ok := addFunc.Type().(*types.Signature)
		if !ok || addSig.Recv() == nil {
			return nil, false
		}

		recv := types.Unalias(addSig.Recv().Type())
		recvPtr, ok := recv.(*types.Pointer)
		if !ok {
			return nil, false
		}
		recvNamed, ok := types.Unalias(recvPtr.Elem()).(*types.Named)
		if !ok {
			return nil, false
		}
		recvObj := recvNamed.Obj()
		if recvObj == nil || recvObj.Pkg() == nil {
			return nil, false
		}
		if recvObj.Pkg().Path() != "math/big" || recvObj.Name() != "Int" {
			return nil, false
		}

		subObj, _, _ := types.LookupFieldOrMethod(recv, false, nil, "Sub")
		subFunc, ok := subObj.(*types.Func)
		if !ok || subFunc == nil {
			return nil, false
		}
		subSig, ok := subFunc.Type().(*types.Signature)
		if !ok {
			return nil, false
		}
		if !types.Identical(addSig, subSig) {
			return nil, false
		}

		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   sel.X,
				Sel: ast.NewIdent("Sub"),
			},
			Lparen:   n.Lparen,
			Args:     append([]ast.Expr(nil), n.Args...),
			Ellipsis: n.Ellipsis,
			Rparen:   n.Rparen,
		}, true
	}, WithName("typed-bigint-add-to-sub"))

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Total == 0 {
		t.Fatalf("total = %d, want > 0", report.Total)
	}
	if report.Killed == 0 {
		t.Fatalf("killed = %d, want > 0 (report=%+v)", report.Killed, report)
	}
	if report.Survived != 0 {
		t.Fatalf("survived = %d, want 0 (report=%+v)", report.Survived, report)
	}
	if report.Errored != 0 {
		t.Fatalf("errored = %d, want 0 (report=%+v)", report.Errored, report)
	}
}
