package graft

import (
	"context"
	"fmt"
	"go/token"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/Warashi/go-graft/internal/execution"
	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutation"
	"github.com/Warashi/go-graft/internal/project"
	"github.com/Warashi/go-graft/internal/rule"
	"github.com/Warashi/go-graft/internal/selection"
)

// Engine is the mutation test framework entry point.
type Engine struct {
	Config Config

	registry *rule.Registry
}

// New creates an engine with validated defaults.
func New(config Config) *Engine {
	return &Engine{
		Config:   config.withDefaults(),
		registry: rule.NewRegistry(),
	}
}

// Run executes mutation testing for package patterns.
func (e *Engine) Run(runCtx context.Context, patterns ...string) (*Report, error) {
	if runCtx == nil {
		runCtx = context.Background()
	}

	registry := e.registry.Snapshot()
	if len(registry.TargetTypes()) == 0 {
		return &Report{}, nil
	}

	prepared, err := e.loadProjectAndSelector(runCtx, patterns...)
	if err != nil {
		return nil, err
	}
	points := mutation.Collect(prepared.project, registry.TargetTypes())
	baseResults, runMutants := e.buildMutants(prepared.workDir, prepared.project, prepared.selector, registry, points)
	runResults := e.runMutants(runCtx, runMutants)
	return composeReport(append(baseResults, runResults...)), nil
}

func compareStrings(a string, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func pointFileSet(pkg *project.Package) *token.FileSet {
	if pkg != nil && pkg.Fset != nil {
		return pkg.Fset
	}
	return token.NewFileSet()
}

func buildImmediateResult(mutantID string, ruleName string, point model.MutationPoint, status model.MutantStatus, reason string) model.MutantExecResult {
	return model.MutantExecResult{
		Mutant: model.Mutant{
			ID:       mutantID,
			RuleName: ruleName,
			Point:    point,
		},
		Status: status,
		Reason: reason,
	}
}

func composeReport(results []model.MutantExecResult) *Report {
	report := &Report{
		Mutants: make([]MutantResult, 0, len(results)),
	}
	summary := execution.Summarize(results)
	report.Total = summary.Total
	report.Killed = summary.Killed
	report.Survived = summary.Survived
	report.Unsupported = summary.Unsupported
	report.Errored = summary.Errored

	for _, res := range results {
		status := mapStatus(res.Status)
		executed := make([]ExecutedPackage, 0, len(res.ExecutedPkgs))
		keys := make([]string, 0, len(res.ExecutedPkgs))
		for pkg := range res.ExecutedPkgs {
			keys = append(keys, pkg)
		}
		slices.Sort(keys)
		for _, pkg := range keys {
			executed = append(executed, ExecutedPackage{
				ImportPath: pkg,
				RunPattern: res.ExecutedPkgs[pkg],
			})
		}

		report.Mutants = append(report.Mutants, MutantResult{
			ID:          res.Mutant.ID,
			RuleName:    res.Mutant.RuleName,
			File:        res.Mutant.Point.FilePath,
			Line:        res.Mutant.Point.Pos.Line,
			Column:      res.Mutant.Point.Pos.Column,
			Package:     res.Mutant.Point.PkgImportPath,
			Executed:    executed,
			Status:      status,
			Reason:      res.Reason,
			Command:     append([]string(nil), res.FailedCommand...),
			Stdout:      res.Stdout,
			Stderr:      res.Stderr,
			TimedOut:    res.TimedOut,
			ElapsedNsec: res.ElapsedNsec,
		})
	}
	return report
}

func mapStatus(status model.MutantStatus) Status {
	switch status {
	case model.MutantKilled:
		return Killed
	case model.MutantSurvived:
		return Survived
	case model.MutantUnsupported:
		return Unsupported
	default:
		return Errored
	}
}

func debugEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("GO_GRAFT_DEBUG")))
	if v == "" {
		return false
	}
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

func mapCallGraphMode(mode TestSelectionCallGraphMode) selection.CallGraphMode {
	switch mode.withDefault() {
	case TestSelectionCallGraphRTA:
		return selection.CallGraphModeRTA
	case TestSelectionCallGraphCHA:
		return selection.CallGraphModeCHA
	case TestSelectionCallGraphAST:
		return selection.CallGraphModeAST
	default:
		return selection.CallGraphModeAuto
	}
}

func writeExcludedMutationTestsDebug(w io.Writer, excluded []selection.ExcludedTest) {
	if w == nil || len(excluded) == 0 {
		return
	}
	items := append([]selection.ExcludedTest(nil), excluded...)
	slices.SortFunc(items, func(a selection.ExcludedTest, b selection.ExcludedTest) int {
		if cmp := compareStrings(a.Ref.ImportPath, b.Ref.ImportPath); cmp != 0 {
			return cmp
		}
		if cmp := compareStrings(a.Ref.Name, b.Ref.Name); cmp != 0 {
			return cmp
		}
		return compareStrings(a.Reason, b.Reason)
	})
	for _, item := range items {
		fmt.Fprintf(w, "go-graft debug: excluded test %s.%s reason=%s\n", item.Ref.ImportPath, item.Ref.Name, item.Reason)
	}
}

func writeTestSelectionCallGraphDebug(w io.Writer, selector *selection.Selector) {
	if w == nil || selector == nil {
		return
	}
	backend := selector.ResolvedBackend()
	if backend == "" {
		return
	}
	fmt.Fprintf(w, "go-graft debug: test selection callgraph backend=%s\n", backend)
	for _, failure := range selector.BuildFailures() {
		fmt.Fprintf(w, "go-graft debug: test selection callgraph fallback=%s\n", failure)
	}
}
