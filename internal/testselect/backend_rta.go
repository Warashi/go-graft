package testselect

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func newRTABackend(project *model.Project, tests []model.TestRef) (*chaBackend, error) {
	if project == nil {
		return nil, fmt.Errorf("project is nil")
	}

	initial := make([]*packages.Package, 0, len(project.Packages))
	ids := make([]string, 0, len(project.Packages))
	fsetByPkg := make(map[string]*token.FileSet, len(project.Packages))
	for _, pkg := range project.Packages {
		if pkg == nil || pkg.Raw == nil {
			continue
		}
		initial = append(initial, pkg.Raw)
		ids = append(ids, pkg.ID)
		fsetByPkg[pkg.ID] = pkg.Fset
	}
	if len(initial) == 0 {
		return nil, fmt.Errorf("no raw packages")
	}

	prog, ssaPkgs := ssautil.AllPackages(initial, ssa.BuilderMode(0))
	if prog == nil {
		return nil, fmt.Errorf("failed to build ssa program")
	}
	prog.Build()

	pkgByID := make(map[string]*ssa.Package, len(ssaPkgs))
	for i, ssaPkg := range ssaPkgs {
		if i >= len(ids) || ssaPkg == nil {
			continue
		}
		pkgByID[ids[i]] = ssaPkg
	}

	testsByFn := make(map[*ssa.Function][]model.TestRef)
	resolvedTests := make(map[*ssa.Function]struct{})
	for _, test := range tests {
		if test.Name == "" {
			continue
		}
		ssaPkg := pkgByID[test.PkgID]
		if ssaPkg == nil {
			continue
		}
		fn := ssaPkg.Func(test.Name)
		if fn == nil {
			continue
		}
		testsByFn[fn] = append(testsByFn[fn], test)
		resolvedTests[fn] = struct{}{}
	}

	rootsSet := make(map[*ssa.Function]struct{})
	for fn := range resolvedTests {
		rootsSet[fn] = struct{}{}
	}
	for pkgID, ssaPkg := range pkgByID {
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

	declToFunc := make(map[functionDeclKey][]*ssa.Function)
	for fn := range ssautil.AllFunctions(prog) {
		if fn == nil {
			continue
		}
		key, ok := declarationKeyFromSSA(fn, prog.Fset)
		if !ok {
			continue
		}
		declToFunc[key] = append(declToFunc[key], fn)
	}

	return &chaBackend{
		backendName: "rta",
		fset:        prog.Fset,
		pkgByID:     pkgByID,
		reverse:     reverse,
		testsByFn:   testsByFn,
		declToFunc:  declToFunc,
		fsetByPkg:   fsetByPkg,
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
