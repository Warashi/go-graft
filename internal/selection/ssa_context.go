package selection

import (
	"fmt"
	"go/token"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/project"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type ssaContext struct {
	prog          *ssa.Program
	pkgByID       map[string]*ssa.Package
	testsByFn     map[*ssa.Function][]model.TestRef
	resolvedTests map[*ssa.Function]struct{}
	declToFunc    map[functionDeclKey][]*ssa.Function
	fsetByPkg     map[string]*token.FileSet
}

func buildSSAContext(project *project.Project, tests []model.TestRef) (*ssaContext, error) {
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

	return &ssaContext{
		prog:          prog,
		pkgByID:       pkgByID,
		testsByFn:     testsByFn,
		resolvedTests: resolvedTests,
		declToFunc:    declToFunc,
		fsetByPkg:     fsetByPkg,
	}, nil
}
