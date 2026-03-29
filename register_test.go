package graft

import (
	"go/ast"
	"go/token"
	"testing"
)

func TestRegisterStoresRulesByConcreteNodeType(t *testing.T) {
	e := New(Config{})

	Register[*ast.BinaryExpr](e, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		n.Op = token.SUB
		return n, true
	}, WithName("binary-sub"), WithDeepCopy())

	Register[*ast.UnaryExpr](e, func(_ *Context, n *ast.UnaryExpr) (*ast.UnaryExpr, bool) {
		n.Op = token.NOT
		return n, true
	})

	snapshot := e.registry.Snapshot()

	binRules := snapshot.RulesFor(&ast.BinaryExpr{})
	if len(binRules) != 1 {
		t.Fatalf("binary rules count = %d, want 1", len(binRules))
	}
	if binRules[0].Name != "binary-sub" {
		t.Fatalf("binary rule name = %q, want %q", binRules[0].Name, "binary-sub")
	}
	if !binRules[0].DeepCopy {
		t.Fatal("binary rule deepCopy = false, want true")
	}

	unRules := snapshot.RulesFor(&ast.UnaryExpr{})
	if len(unRules) != 1 {
		t.Fatalf("unary rules count = %d, want 1", len(unRules))
	}
	if unRules[0].Name != "rule#1" {
		t.Fatalf("default rule name = %q, want %q", unRules[0].Name, "rule#1")
	}
	if unRules[0].DeepCopy {
		t.Fatal("unary rule deepCopy = true, want false")
	}
}

func TestRegisterPanicsOnInvalidArguments(t *testing.T) {
	t.Run("nil engine", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		Register[*ast.BinaryExpr](nil, func(_ *Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
			return n, true
		})
	})

	t.Run("nil mutate", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic")
			}
		}()
		Register[*ast.BinaryExpr](New(Config{}), nil)
	})
}
