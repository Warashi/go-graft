package graft

import (
	"context"
	"go/ast"
	"go/token"
	"os"
	"testing"
	"time"
)

const dogfoodChildEnv = "GO_GRAFT_DOGFOOD_CHILD"

func TestEngineRunDogfoodRepository(t *testing.T) {
	if os.Getenv(dogfoodChildEnv) == "1" {
		t.Skip("skip dogfood test in child go test process")
	}
	t.Setenv(dogfoodChildEnv, "1")

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
}
