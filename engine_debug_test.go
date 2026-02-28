package graft

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/testdiscover"
	"github.com/Warashi/go-graft/internal/testselect"
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

func TestWriteTestSelectionCallGraphDebug(t *testing.T) {
	project := &model.Project{
		Packages: []*model.Package{
			{
				ID:         "p",
				ImportPath: "example.com/p",
			},
		},
	}
	tests := []model.TestRef{
		{
			PkgID:      "p",
			ImportPath: "example.com/p",
			Name:       "TestP",
		},
	}
	selector := testselect.NewSelectorWithOptions(project, tests, testselect.SelectorOptions{
		CallGraphMode: testselect.CallGraphModeRTA,
	})

	var buf bytes.Buffer
	writeTestSelectionCallGraphDebug(&buf, selector)
	got := buf.String()
	if !strings.Contains(got, "go-graft debug: test selection callgraph backend=ast\n") {
		t.Fatalf("missing backend line in %q", got)
	}
	if !strings.Contains(got, "go-graft debug: test selection callgraph fallback=rta:") {
		t.Fatalf("missing rta fallback line in %q", got)
	}
	if !strings.Contains(got, "go-graft debug: test selection callgraph fallback=cha:") {
		t.Fatalf("missing cha fallback line in %q", got)
	}
}
