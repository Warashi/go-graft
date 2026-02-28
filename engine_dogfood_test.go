package graft

import (
	"context"
	"go/ast"
	"go/token"
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
