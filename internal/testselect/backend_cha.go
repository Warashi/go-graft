package testselect

import (
	"fmt"
	"go/token"
	"path/filepath"

	"github.com/Warashi/go-graft/internal/model"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type chaBackend struct {
	backendName string
	fset       *token.FileSet
	pkgByID    map[string]*ssa.Package
	reverse    map[*ssa.Function][]*ssa.Function
	testsByFn  map[*ssa.Function][]model.TestRef
	declToFunc map[functionDeclKey][]*ssa.Function
	fsetByPkg  map[string]*token.FileSet
}

type functionDeclKey struct {
	filePath string
	line     int
	name     string
}

func newCHABackend(project *model.Project, tests []model.TestRef) (*chaBackend, error) {
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

	cg := cha.CallGraph(prog)
	if cg == nil {
		return nil, fmt.Errorf("cha returned nil call graph")
	}
	reverse := buildReverseFromCallGraph(cg)

	testsByFn := make(map[*ssa.Function][]model.TestRef)
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

	return &chaBackend{
		backendName: "cha",
		fset:       prog.Fset,
		pkgByID:    pkgByID,
		reverse:    reverse,
		testsByFn:  testsByFn,
		declToFunc: declToFunc,
		fsetByPkg:  fsetByPkg,
	}, nil
}

func (b *chaBackend) name() string {
	return b.backendName
}

func (b *chaBackend) candidateTests(point model.MutationPoint) []model.TestRef {
	if b == nil {
		return nil
	}
	seeds := b.resolveSeeds(point)
	if len(seeds) == 0 {
		return nil
	}

	seenFuncs := make(map[*ssa.Function]struct{}, len(seeds))
	queue := make([]*ssa.Function, 0, len(seeds))
	for _, seed := range seeds {
		if seed == nil {
			continue
		}
		if _, ok := seenFuncs[seed]; ok {
			continue
		}
		seenFuncs[seed] = struct{}{}
		queue = append(queue, seed)
	}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, caller := range b.reverse[cur] {
			if caller == nil {
				continue
			}
			if _, ok := seenFuncs[caller]; ok {
				continue
			}
			seenFuncs[caller] = struct{}{}
			queue = append(queue, caller)
		}
	}

	out := make([]model.TestRef, 0)
	seenTests := make(map[functionKey]struct{})
	for fn := range seenFuncs {
		tests := b.testsByFn[fn]
		for _, test := range tests {
			key := functionKey{pkgID: test.PkgID, name: test.Name}
			if _, ok := seenTests[key]; ok {
				continue
			}
			seenTests[key] = struct{}{}
			out = append(out, test)
		}
	}
	return out
}

func (b *chaBackend) resolveSeeds(point model.MutationPoint) []*ssa.Function {
	if point.EnclosingFunc == nil || point.EnclosingFunc.Name == nil {
		return nil
	}
	if len(point.Path) == 0 {
		return nil
	}

	out := make([]*ssa.Function, 0, 4)
	seen := make(map[*ssa.Function]struct{})
	add := func(fn *ssa.Function) {
		if fn == nil {
			return
		}
		if _, ok := seen[fn]; ok {
			return
		}
		seen[fn] = struct{}{}
		out = append(out, fn)
	}

	ssaPkg := b.pkgByID[point.PkgID]
	if ssaPkg != nil {
		add(ssa.EnclosingFunction(ssaPkg, point.Path))
		add(ssaPkg.Func(point.EnclosingFunc.Name.Name))
	}

	if key, ok := b.declarationKeyFromMutationPoint(point); ok {
		for _, fn := range b.declToFunc[key] {
			add(fn)
		}
	}
	return out
}

func (b *chaBackend) declarationKeyFromMutationPoint(point model.MutationPoint) (functionDeclKey, bool) {
	if point.EnclosingFunc == nil || point.EnclosingFunc.Name == nil {
		return functionDeclKey{}, false
	}
	fset := b.fsetByPkg[point.PkgID]
	if fset == nil {
		return functionDeclKey{}, false
	}
	pos := fset.Position(point.EnclosingFunc.Name.Pos())
	if pos.Filename == "" || pos.Line <= 0 {
		return functionDeclKey{}, false
	}
	return functionDeclKey{
		filePath: filepath.Clean(pos.Filename),
		line:     pos.Line,
		name:     point.EnclosingFunc.Name.Name,
	}, true
}

func declarationKeyFromSSA(fn *ssa.Function, fset *token.FileSet) (functionDeclKey, bool) {
	if fn == nil || fset == nil {
		return functionDeclKey{}, false
	}
	pos := fset.Position(fn.Pos())
	if pos.Filename == "" || pos.Line <= 0 {
		return functionDeclKey{}, false
	}
	return functionDeclKey{
		filePath: filepath.Clean(pos.Filename),
		line:     pos.Line,
		name:     fn.Name(),
	}, true
}

func buildReverseFromCallGraph(cg *callgraph.Graph) map[*ssa.Function][]*ssa.Function {
	reverseSet := make(map[*ssa.Function]map[*ssa.Function]struct{})
	for _, node := range cg.Nodes {
		if node == nil || node.Func == nil {
			continue
		}
		for _, edge := range node.Out {
			if edge == nil || edge.Callee == nil || edge.Callee.Func == nil || edge.Caller == nil || edge.Caller.Func == nil {
				continue
			}
			callee := edge.Callee.Func
			caller := edge.Caller.Func
			if _, ok := reverseSet[callee]; !ok {
				reverseSet[callee] = make(map[*ssa.Function]struct{})
			}
			reverseSet[callee][caller] = struct{}{}
		}
	}
	reverse := make(map[*ssa.Function][]*ssa.Function, len(reverseSet))
	for callee, callers := range reverseSet {
		for caller := range callers {
			reverse[callee] = append(reverse[callee], caller)
		}
	}
	return reverse
}
