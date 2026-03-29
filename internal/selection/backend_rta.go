package selection

import (
	"fmt"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/project"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

func newRTABackend(project *project.Project, tests []model.TestRef) (*chaBackend, error) {
	ctx, err := buildSSAContext(project, tests)
	if err != nil {
		return nil, err
	}

	rootsSet := make(map[*ssa.Function]struct{})
	for fn := range ctx.resolvedTests {
		rootsSet[fn] = struct{}{}
	}
	for pkgID, ssaPkg := range ctx.pkgByID {
		if ssaPkg == nil {
			continue
		}
		if fn := ssaPkg.Func("init"); fn != nil {
			rootsSet[fn] = struct{}{}
		}
		if strings.HasSuffix(pkgID, ".test") {
			if fn := ssaPkg.Func("main"); fn != nil {
				rootsSet[fn] = struct{}{}
			}
		}
	}
	if len(rootsSet) == 0 {
		return nil, fmt.Errorf("no rta roots")
	}

	roots := make([]*ssa.Function, 0, len(rootsSet))
	for fn := range rootsSet {
		roots = append(roots, fn)
	}
	result, err := analyzeRTASafely(roots)
	if err != nil {
		return nil, err
	}
	if result == nil || result.CallGraph == nil {
		return nil, fmt.Errorf("rta returned nil call graph")
	}
	reverse := buildReverseFromCallGraph(result.CallGraph)

	return &chaBackend{
		backendName: "rta",
		fset:        ctx.prog.Fset,
		pkgByID:     ctx.pkgByID,
		reverse:     reverse,
		testsByFn:   ctx.testsByFn,
		declToFunc:  ctx.declToFunc,
		fsetByPkg:   ctx.fsetByPkg,
	}, nil
}

func analyzeRTASafely(roots []*ssa.Function) (_ *rta.Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("rta panicked: %v", recovered)
		}
	}()
	return rta.Analyze(roots, true), nil
}
