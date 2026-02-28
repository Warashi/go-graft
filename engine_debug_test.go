package graft

import (
	"bytes"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/testdiscover"
)

func TestDebugEnabled(t *testing.T) {
	t.Setenv("GO_GRAFT_DEBUG", "")
	if debugEnabled() {
		t.Fatal("debugEnabled() should be false for empty value")
	}

	t.Setenv("GO_GRAFT_DEBUG", "0")
	if debugEnabled() {
		t.Fatal("debugEnabled() should be false for 0")
	}

	t.Setenv("GO_GRAFT_DEBUG", "false")
	if debugEnabled() {
		t.Fatal("debugEnabled() should be false for false")
	}

	t.Setenv("GO_GRAFT_DEBUG", "1")
	if !debugEnabled() {
		t.Fatal("debugEnabled() should be true for 1")
	}
}

func TestWriteExcludedMutationTestsDebug(t *testing.T) {
	excluded := []testdiscover.ExcludedTest{
		{
			Ref: model.TestRef{
				ImportPath: "example.com/b",
				Name:       "TestB",
			},
			Reason: testdiscover.ExcludeReasonDirectiveExclude,
		},
		{
			Ref: model.TestRef{
				ImportPath: "example.com/a",
				Name:       "TestA",
			},
			Reason: testdiscover.ExcludeReasonAutoRunReachable,
		},
	}

	var buf bytes.Buffer
	writeExcludedMutationTestsDebug(&buf, excluded)
	got := buf.String()
	want := "go-graft debug: excluded test example.com/a.TestA reason=auto-run-reachable\n" +
		"go-graft debug: excluded test example.com/b.TestB reason=directive-exclude\n"
	if got != want {
		t.Fatalf("writeExcludedMutationTestsDebug() = %q, want %q", got, want)
	}
}
