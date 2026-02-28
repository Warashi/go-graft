package dogfoodcalc

import "testing"

func TestAddInt64(t *testing.T) {
	if got := AddInt64(10, 4); got != 14 {
		t.Fatalf("AddInt64(10, 4) = %d, want 14", got)
	}
}

func TestAddInt64ViaIdent(t *testing.T) {
	if got := AddInt64ViaIdent(10, 4); got != 14 {
		t.Fatalf("AddInt64ViaIdent(10, 4) = %d, want 14", got)
	}
}

func TestAbsFloat64(t *testing.T) {
	if got := AbsFloat64(-2.5); got != 2.5 {
		t.Fatalf("AbsFloat64(-2.5) = %v, want 2.5", got)
	}
}

func TestMinInt(t *testing.T) {
	if got := MinInt([]int{7, 1, 5}); got != 1 {
		t.Fatalf("MinInt(...) = %d, want 1", got)
	}
}
