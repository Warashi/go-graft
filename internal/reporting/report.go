package reporting

import "github.com/Warashi/go-graft/internal/model"

type Summary struct {
	Total       int
	Killed      int
	Survived    int
	Unsupported int
	Errored     int
}

func Summarize(results []model.MutantExecResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		switch result.Status {
		case model.MutantKilled:
			summary.Killed++
		case model.MutantSurvived:
			summary.Survived++
		case model.MutantUnsupported:
			summary.Unsupported++
		case model.MutantErrored:
			summary.Errored++
		}
	}
	return summary
}
