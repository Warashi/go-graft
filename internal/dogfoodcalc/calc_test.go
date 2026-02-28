package dogfoodcalc

import "testing"

func TestAddInt64(t *testing.T) {
	if got := AddInt64(10, 4); got != 14 {
		t.Fatalf("AddInt64(10, 4) = %d, want 14", got)
	}
}
