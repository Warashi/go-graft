package graft

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/Warashi/go-graft/internal/astcow"
	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutantbuild"
	"github.com/Warashi/go-graft/internal/mutationpoint"
	"github.com/Warashi/go-graft/internal/projectload"
	"github.com/Warashi/go-graft/internal/reporting"
	"github.com/Warashi/go-graft/internal/runner"
	"github.com/Warashi/go-graft/internal/testdiscover"
	"github.com/Warashi/go-graft/internal/testselect"
)

// Engine is the mutation test framework entry point.
type Engine struct {
	Config Config

	mu       sync.RWMutex
	registry *ruleRegistry
}

// New creates an engine with validated defaults.
func New(config Config) *Engine {
	return &Engine{
		Config:   config.withDefaults(),
		registry: newRuleRegistry(),
	}
}

// Run executes mutation testing for package patterns.
func (e *Engine) Run(runCtx context.Context, patterns ...string) (*Report, error) {
	if runCtx == nil {
		runCtx = context.Background()
	}

	registry := e.snapshotRegistry()
	if len(registry.ordered) == 0 {
		return &Report{}, nil
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("graft: getwd: %w", err)
	}

	project, err := (projectload.Loader{Dir: workDir}).Load(runCtx, patterns...)
	if err != nil {
		return nil, err
	}
	discovered := testdiscover.DiscoverDetailed(project)
	if debugEnabled() {
		writeExcludedMutationTestsDebug(os.Stderr, discovered.Excluded)
	}
	tests := discovered.Included
	points := mutationpoint.Collect(project, registry.targetTypes())

	builder := mutantbuild.Builder{BaseTempDir: e.Config.BaseTempDir}
	baseResults := make([]model.MutantExecResult, 0)
	runMutants := make([]model.Mutant, 0)
	mutantSeq := 1

	for _, point := range points {
		rules := registry.byType[reflect.TypeOf(point.Node)]
		if len(rules) == 0 {
			continue
		}
		pkg := project.ByID[point.PkgID]
		fset := pointFileSet(pkg)

		for _, rule := range rules {
			mutantID := "m-" + itoa(mutantSeq)
			mutantSeq++

			nodeInput := astcow.ShallowCopyNode(point.Node)
			if nodeInput == nil {
				baseResults = append(baseResults, buildImmediateResult(mutantID, rule.name, point, model.MutantErrored, "unsupported mutation node type"))
				continue
			}

			callbackCtx := newContext()
			if pkg != nil {
				callbackCtx.Fset = pkg.Fset
				callbackCtx.Pkg = pkg.Raw
				callbackCtx.Types = pkg.TypesInfo
			}
			callbackCtx.File = point.File
			callbackCtx.Path = append([]ast.Node(nil), point.Path...)
			callbackCtx.setOriginal(nodeInput, point.Node)

			mutatedNode, changed, mutateErr := applyRule(rule, callbackCtx, nodeInput)
			if mutateErr != nil {
				baseResults = append(baseResults, buildImmediateResult(mutantID, rule.name, point, model.MutantErrored, mutateErr.Error()))
				continue
			}
			if !changed {
				continue
			}

			fileMut, cloneMap, err := astcow.ClonePath(point.Path, point.Node, mutatedNode)
			if err != nil {
				baseResults = append(baseResults, buildImmediateResult(mutantID, rule.name, point, model.MutantErrored, err.Error()))
				continue
			}
			callbackCtx.cloneMap = cloneMap

			mutant, err := builder.Build(mutantbuild.Input{
				ID:         mutantID,
				RuleName:   rule.name,
				Point:      point,
				Mutated:    fileMut,
				Fset:       fset,
				BaseTempID: mutantID,
			})
			if err != nil {
				baseResults = append(baseResults, buildImmediateResult(mutantID, rule.name, point, model.MutantErrored, err.Error()))
				continue
			}
			mutant.WorkDir = workDir
			mutant.SelectedTests = testselect.Select(project, tests, point)
			runMutants = append(runMutants, *mutant)
		}
	}

	runResults := runner.Runner{
		Workers:       e.Config.Workers,
		MutantTimeout: e.Config.MutantTimeout,
		KeepTemp:      e.Config.KeepTemp,
	}.Run(runCtx, runMutants)

	allResults := append(baseResults, runResults...)
	return composeReport(allResults), nil
}

func (e *Engine) snapshotRegistry() ruleRegistry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := ruleRegistry{
		ordered: append([]registeredRule(nil), e.registry.ordered...),
		byType:  make(map[reflect.Type][]registeredRule, len(e.registry.byType)),
	}
	for key, rules := range e.registry.byType {
		out.byType[key] = append([]registeredRule(nil), rules...)
	}
	return out
}

func (r ruleRegistry) targetTypes() []reflect.Type {
	types := make([]reflect.Type, 0, len(r.byType))
	for t := range r.byType {
		types = append(types, t)
	}
	slices.SortFunc(types, func(a reflect.Type, b reflect.Type) int {
		return compareStrings(a.String(), b.String())
	})
	return types
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

func applyRule(rule registeredRule, ctx *Context, node ast.Node) (mutated ast.Node, changed bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("rule %q panicked: %v", rule.name, recovered)
		}
	}()
	mutated, changed = rule.mutate(ctx, node)
	if changed && mutated == nil {
		return nil, false, fmt.Errorf("rule %q returned nil mutant node", rule.name)
	}
	return mutated, changed, nil
}

func pointFileSet(pkg *model.Package) *token.FileSet {
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
	summary := reporting.Summarize(results)
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

func writeExcludedMutationTestsDebug(w io.Writer, excluded []testdiscover.ExcludedTest) {
	if w == nil || len(excluded) == 0 {
		return
	}
	items := append([]testdiscover.ExcludedTest(nil), excluded...)
	slices.SortFunc(items, func(a testdiscover.ExcludedTest, b testdiscover.ExcludedTest) int {
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
