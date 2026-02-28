package graft

import (
	"context"
	"math/big"
	"testing"
	"time"
)

func TestRegisterMethodCallSwapPanicsOnNilEngine(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	RegisterMethodCallSwap(nil, (*big.Int).Add, (*big.Int).Sub)
}

func TestRegisterMethodCallSwapPanicsOnNonFunc(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	RegisterMethodCallSwap(New(Config{}), 1, 1)
}

func TestRegisterMethodCallSwapPanicsOnMethodValue(t *testing.T) {
	var z big.Int

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	RegisterMethodCallSwap(New(Config{}), z.Add, z.Sub)
}

func TestRegisterMethodCallSwapDogfoodBigIntAddToSub(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterMethodCallSwap(e, (*big.Int).Add, (*big.Int).Sub, WithName("typed-bigint-add-to-sub"))

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

func TestRegisterMethodCallSwapSkipsWhenReceiverMismatch(t *testing.T) {
	e := New(Config{
		Workers:       1,
		MutantTimeout: 30 * time.Second,
	})
	RegisterMethodCallSwap(e, (*big.Rat).Add, (*big.Rat).Sub, WithName("typed-bigrat-add-to-sub"))

	report, err := e.Run(context.Background(), "./internal/dogfoodcalc")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.Total != 0 {
		t.Fatalf("total = %d, want 0 (report=%+v)", report.Total, report)
	}
}
