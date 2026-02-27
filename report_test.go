package graft

import "testing"

func TestReportMutationScore(t *testing.T) {
	r := Report{Killed: 3, Survived: 1, Unsupported: 10}
	got := r.MutationScore()
	if got != 0.75 {
		t.Fatalf("MutationScore() = %v, want %v", got, 0.75)
	}
}

func TestReportMutationScoreWithNoScoredMutants(t *testing.T) {
	r := Report{Unsupported: 4}
	got := r.MutationScore()
	if got != 0 {
		t.Fatalf("MutationScore() = %v, want 0", got)
	}
}
