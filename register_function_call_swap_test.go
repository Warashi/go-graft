package graft

import (
	"context"
	"math"
	"math/big"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Warashi/go-graft/internal/dogfoodcalc"
)

func testFunctionCallSwapIdentity(v int) int {
	return v
}

func testFunctionCallSwapIdentityAlt(v int) int {
	return v
}

func TestRegisterFunctionCallSwapPanicsOnNilEngine(t *testing.T) {
	assertPanics(t, func() {
		RegisterFunctionCallSwap(nil, math.Abs, math.Ceil)
	})
}

func TestRegisterFunctionCallSwapPanicsOnNonFunc(t *testing.T) {
	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), 1, 1)
	})
}

func TestRegisterFunctionCallSwapPanicsOnNilFunction(t *testing.T) {
	var from func(int) int

	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), from, testFunctionCallSwapIdentity)
	})
}

func TestRegisterFunctionCallSwapPanicsOnClosure(t *testing.T) {
	closure := func(v int) int {
		return v
	}

	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), closure, testFunctionCallSwapIdentityAlt)
	})
}

func TestRegisterFunctionCallSwapPanicsOnMethodExpression(t *testing.T) {
	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), (*big.Int).Add, (*big.Int).Sub)
	})
}

func TestRegisterFunctionCallSwapPanicsOnMethodValue(t *testing.T) {
	var z big.Int

	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), z.Add, z.Sub)
	})
}

func TestRegisterFunctionCallSwapPanicsOnDifferentPackages(t *testing.T) {
	assertPanics(t, func() {
		RegisterFunctionCallSwap(New(Config{}), strings.TrimSpace, path.Clean)
	})
}

func TestRegisterFunctionCallSwapDogfoodIdent(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterFunctionCallSwap(e,
		dogfoodcalc.MakeBigIntFromInt64,
		dogfoodcalc.MakeNegatedBigIntFromInt64,
		WithName("typed-ident-bigint-positive-to-negative"),
	)

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertReportKilledOnly(t, report)
}

func TestRegisterFunctionCallSwapDogfoodSelector(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterFunctionCallSwap(e, math.Abs, math.Ceil, WithName("typed-math-abs-to-ceil"))

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertReportKilledOnly(t, report)
}

func TestRegisterFunctionCallSwapDogfoodGeneric(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterFunctionCallSwap(e,
		slices.Min[[]int, int],
		slices.Max[[]int, int],
		WithName("typed-slices-min-to-max"),
	)

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	assertReportKilledOnly(t, report)
}

func TestRegisterFunctionCallSwapSkipsWhenFromDoesNotAppear(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterFunctionCallSwap(e, strings.TrimSpace, strings.ToLower, WithName("typed-strings-trim-to-lower"))

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Total != 0 {
		t.Fatalf("total = %d, want 0 (report=%+v)", report.Total, report)
	}
}

func assertPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}

func assertReportKilledOnly(t *testing.T, report *Report) {
	t.Helper()
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
