package graft

import (
	"context"
	"errors"
	"go/ast"
	"os"
	"reflect"

	"github.com/Warashi/go-graft/internal/astcow"
	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutantbuild"
	"github.com/Warashi/go-graft/internal/project"
	"github.com/Warashi/go-graft/internal/runner"
	"github.com/Warashi/go-graft/internal/testdiscover"
	"github.com/Warashi/go-graft/internal/testselect"
)

var errUnsupportedMutationNodeType = errors.New("unsupported mutation node type")

type runPreparation struct {
	workDir  string
	project  *project.Project
	selector *testselect.Selector
}

func (e *Engine) loadProjectAndSelector(runCtx context.Context, patterns ...string) (runPreparation, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return runPreparation{}, err
	}

	project, err := (project.Loader{Dir: workDir}).Load(runCtx, patterns...)
	if err != nil {
		return runPreparation{}, err
	}
	discovered := testdiscover.DiscoverDetailed(project)
	if debugEnabled() {
		writeExcludedMutationTestsDebug(os.Stderr, discovered.Excluded)
	}
	tests := discovered.Included
	selector := testselect.NewSelectorWithOptions(project, tests, testselect.SelectorOptions{
		CallGraphMode: mapCallGraphMode(e.Config.TestSelectionCallGraph),
	})
	if debugEnabled() {
		writeTestSelectionCallGraphDebug(os.Stderr, selector)
	}

	return runPreparation{
		workDir:  workDir,
		project:  project,
		selector: selector,
	}, nil
}

func (e *Engine) buildMutants(workDir string, project *project.Project, selector *testselect.Selector, registry ruleRegistry, points []model.MutationPoint) ([]model.MutantExecResult, []model.Mutant) {
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

			callbackCtx := newMutationContext(pkg, point)
			nodeInput, err := prepareRuleInput(rule, point, callbackCtx)
			if err != nil {
				appendImmediateErrored(&baseResults, mutantID, rule.name, point, err.Error())
				continue
			}

			mutatedNode, changed, mutateErr := applyRule(rule, callbackCtx, nodeInput)
			if mutateErr != nil {
				appendImmediateErrored(&baseResults, mutantID, rule.name, point, mutateErr.Error())
				continue
			}
			if !changed {
				continue
			}

			fileMut, cloneMap, err := astcow.ClonePath(point.Path, point.Node, mutatedNode)
			if err != nil {
				appendImmediateErrored(&baseResults, mutantID, rule.name, point, err.Error())
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
				appendImmediateErrored(&baseResults, mutantID, rule.name, point, err.Error())
				continue
			}
			mutant.WorkDir = workDir
			mutant.SelectedTests = selector.Select(point)
			runMutants = append(runMutants, *mutant)
		}
	}
	return baseResults, runMutants
}

func (e *Engine) runMutants(runCtx context.Context, runMutants []model.Mutant) []model.MutantExecResult {
	return runner.Runner{
		Workers:       e.Config.Workers,
		MutantTimeout: e.Config.MutantTimeout,
		KeepTemp:      e.Config.KeepTemp,
	}.Run(runCtx, runMutants)
}

func newMutationContext(pkg *project.Package, point model.MutationPoint) *Context {
	callbackCtx := newContext()
	if pkg != nil {
		callbackCtx.Fset = pkg.Fset
		callbackCtx.Pkg = pkg.Raw
		callbackCtx.Types = pkg.TypesInfo
	}
	callbackCtx.File = point.File
	callbackCtx.Path = append([]ast.Node(nil), point.Path...)
	return callbackCtx
}

func prepareRuleInput(rule registeredRule, point model.MutationPoint, callbackCtx *Context) (ast.Node, error) {
	if rule.deepCopy {
		deepCopied, cloneMap, err := astcow.DeepCopyNode(point.Node)
		if err != nil {
			return nil, err
		}
		for clone, original := range cloneMap {
			callbackCtx.setOriginal(clone, original)
		}
		return deepCopied, nil
	}

	nodeInput := astcow.ShallowCopyNode(point.Node)
	if nodeInput == nil {
		return nil, errUnsupportedMutationNodeType
	}
	callbackCtx.setOriginal(nodeInput, point.Node)
	return nodeInput, nil
}

func appendImmediateErrored(results *[]model.MutantExecResult, mutantID string, ruleName string, point model.MutationPoint, reason string) {
	*results = append(*results, buildImmediateResult(mutantID, ruleName, point, model.MutantErrored, reason))
}
