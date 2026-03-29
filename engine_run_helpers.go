package graft

import (
	"context"
	"errors"
	"go/ast"
	"os"
	"strconv"

	"github.com/Warashi/go-graft/internal/astclone"
	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/mutation"
	"github.com/Warashi/go-graft/internal/project"
	"github.com/Warashi/go-graft/internal/rule"
	"github.com/Warashi/go-graft/internal/runner"
	"github.com/Warashi/go-graft/internal/selection"
)

var errUnsupportedMutationNodeType = errors.New("unsupported mutation node type")

type runPreparation struct {
	workDir  string
	project  *project.Project
	selector *selection.Selector
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
	discovered := selection.DiscoverDetailed(project)
	if debugEnabled() {
		writeExcludedMutationTestsDebug(os.Stderr, discovered.Excluded)
	}
	tests := discovered.Included
	selector := selection.NewSelectorWithOptions(project, tests, selection.SelectorOptions{
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

func (e *Engine) buildMutants(workDir string, project *project.Project, selector *selection.Selector, registry rule.Snapshot, points []model.MutationPoint) ([]model.MutantExecResult, []model.Mutant) {
	builder := mutation.Builder{BaseTempDir: e.Config.BaseTempDir}
	baseResults := make([]model.MutantExecResult, 0)
	runMutants := make([]model.Mutant, 0)
	mutantSeq := 1

	for _, point := range points {
		rules := registry.RulesFor(point.Node)
		if len(rules) == 0 {
			continue
		}
		pkg := project.ByID[point.PkgID]
		fset := pointFileSet(pkg)

		for _, ruleDef := range rules {
			mutantID := "m-" + strconv.Itoa(mutantSeq)
			mutantSeq++

			callbackCtx := newMutationContext(pkg, point)
			nodeInput, err := prepareRuleInput(ruleDef, point, callbackCtx)
			if err != nil {
				appendImmediateErrored(&baseResults, mutantID, ruleDef.Name, point, err.Error())
				continue
			}

			mutatedNode, changed, mutateErr := rule.Apply(ruleDef, callbackCtx, nodeInput)
			if mutateErr != nil {
				appendImmediateErrored(&baseResults, mutantID, ruleDef.Name, point, mutateErr.Error())
				continue
			}
			if !changed {
				continue
			}

			fileMut, cloneMap, err := astclone.ClonePath(point.Path, point.Node, mutatedNode)
			if err != nil {
				appendImmediateErrored(&baseResults, mutantID, ruleDef.Name, point, err.Error())
				continue
			}
			for clone, original := range cloneMap {
				callbackCtx.SetOriginal(clone, original)
			}

			mutant, err := builder.Build(mutation.Input{
				ID:         mutantID,
				RuleName:   ruleDef.Name,
				Point:      point,
				Mutated:    fileMut,
				Fset:       fset,
				BaseTempID: mutantID,
			})
			if err != nil {
				appendImmediateErrored(&baseResults, mutantID, ruleDef.Name, point, err.Error())
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

func prepareRuleInput(def rule.Definition, point model.MutationPoint, callbackCtx *Context) (ast.Node, error) {
	if def.DeepCopy {
		deepCopied, cloneMap, err := astclone.DeepCopyNode(point.Node)
		if err != nil {
			return nil, err
		}
		for clone, original := range cloneMap {
			callbackCtx.SetOriginal(clone, original)
		}
		return deepCopied, nil
	}

	nodeInput := astclone.ShallowCopyNode(point.Node)
	if nodeInput == nil {
		return nil, errUnsupportedMutationNodeType
	}
	callbackCtx.SetOriginal(nodeInput, point.Node)
	return nodeInput, nil
}

func appendImmediateErrored(results *[]model.MutantExecResult, mutantID string, ruleName string, point model.MutationPoint, reason string) {
	*results = append(*results, buildImmediateResult(mutantID, ruleName, point, model.MutantErrored, reason))
}
