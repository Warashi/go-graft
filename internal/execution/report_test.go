package execution

import (
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestSummarize(t *testing.T) {
	results := []model.MutantExecResult{
		{Status: model.MutantKilled},
		{Status: model.MutantSurvived},
		{Status: model.MutantUnsupported},
		{Status: model.MutantErrored},
		{Status: model.MutantKilled},
	}
	got := Summarize(results)
	if got.Total != 5 || got.Killed != 2 || got.Survived != 1 || got.Unsupported != 1 || got.Errored != 1 {
		t.Fatalf("summary = %+v", got)
	}
}
