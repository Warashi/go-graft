package testselect

import (
	"go/ast"
	"path/filepath"
	"slices"

	"github.com/Warashi/go-graft/internal/model"
)

type functionKey struct {
	pkgID string
	name  string
}

// Selector picks tests for mutation points in one loaded project.
type Selector struct {
	project       *model.Project
	tests         []model.TestRef
	backend       selectorBackend
	backendName   string
	buildFailures []string
}

// NewSelector creates a selector and caches AST-level reverse callers.
func NewSelector(project *model.Project, tests []model.TestRef) *Selector {
	return NewSelectorWithOptions(project, tests, SelectorOptions{
		CallGraphMode: CallGraphModeAST,
	})
}

// Select picks tests for one mutation point.
func Select(project *model.Project, tests []model.TestRef, point model.MutationPoint) model.SelectedTests {
	return NewSelector(project, tests).Select(point)
}

// Select picks tests for one mutation point.
func (s *Selector) Select(point model.MutationPoint) model.SelectedTests {
	selected := model.SelectedTests{
		ByImportPath: make(map[string][]string),
	}
	if s == nil || s.project == nil || len(s.tests) == 0 {
		return selected
	}

	allowedPkgs := reverseDependers(s.project, point.PkgImportPath)
	candidate := s.backend.candidateTests(point)
	if len(candidate) == 0 {
		candidate = append([]model.TestRef(nil), s.tests...)
	}

	unique := make(map[string]map[string]struct{})
	for _, test := range candidate {
		if _, ok := allowedPkgs[test.ImportPath]; !ok {
			continue
		}
		if _, ok := unique[test.ImportPath]; !ok {
			unique[test.ImportPath] = make(map[string]struct{})
		}
		unique[test.ImportPath][test.Name] = struct{}{}
	}

	for pkg, namesSet := range unique {
		names := make([]string, 0, len(namesSet))
		for name := range namesSet {
			names = append(names, name)
		}
		slices.Sort(names)
		selected.ByImportPath[pkg] = names
	}
	return selected
}

func candidateTestsByReachability(tests []model.TestRef, point model.MutationPoint, reverseCallers map[functionKey][]functionKey) []model.TestRef {
	if point.EnclosingFunc == nil || point.EnclosingFunc.Name == nil {
		return append([]model.TestRef(nil), tests...)
	}

	seed := functionKey{pkgID: point.PkgID, name: point.EnclosingFunc.Name.Name}
	seen := map[functionKey]struct{}{
		seed: {},
	}
	queue := []functionKey{seed}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, caller := range reverseCallers[cur] {
			if _, ok := seen[caller]; ok {
				continue
			}
			seen[caller] = struct{}{}
			queue = append(queue, caller)
		}
	}

	out := make([]model.TestRef, 0)
	for _, test := range tests {
		if _, ok := seen[functionKey{pkgID: test.PkgID, name: test.Name}]; ok {
			out = append(out, test)
		}
	}
	return out
}

func buildReverseCallers(project *model.Project) map[functionKey][]functionKey {
	if project == nil {
		return nil
	}
	funcBodies := make(map[functionKey]*ast.BlockStmt)
	importAliases := make(map[string]map[string]string) // pkgID -> alias -> importPath

	for _, pkg := range project.Packages {
		if pkg == nil {
			continue
		}
		for _, filePath := range pkg.GoFiles {
			file := pkg.SyntaxByPath[filePath]
			if file == nil {
				continue
			}
			if _, ok := importAliases[pkg.ID]; !ok {
				importAliases[pkg.ID] = map[string]string{}
			}
			aliasMap := importAliases[pkg.ID]
			for _, imp := range file.Imports {
				if imp.Path == nil {
					continue
				}
				importPath := trimQuotes(imp.Path.Value)
				if importPath == "" {
					continue
				}
				alias := filepath.Base(importPath)
				if imp.Name != nil && imp.Name.Name != "." && imp.Name.Name != "_" {
					alias = imp.Name.Name
				}
				aliasMap[alias] = importPath
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Body == nil {
					continue
				}
				funcBodies[functionKey{pkgID: pkg.ID, name: fn.Name.Name}] = fn.Body
			}
		}
	}

	byImport := map[string][]string{}
	for _, pkg := range project.Packages {
		byImport[pkg.ImportPath] = append(byImport[pkg.ImportPath], pkg.ID)
	}

	reverse := make(map[functionKey][]functionKey)
	for caller, body := range funcBodies {
		aliases := importAliases[caller.pkgID]
		ast.Inspect(body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			callee, ok := resolveCall(caller.pkgID, call, aliases, byImport)
			if !ok {
				return true
			}
			reverse[callee] = append(reverse[callee], caller)
			return true
		})
	}
	return reverse
}

func resolveCall(currentPkgID string, call *ast.CallExpr, aliases map[string]string, byImport map[string][]string) (functionKey, bool) {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return functionKey{pkgID: currentPkgID, name: fun.Name}, true
	case *ast.SelectorExpr:
		pkgIdent, ok := fun.X.(*ast.Ident)
		if !ok {
			return functionKey{}, false
		}
		importPath, ok := aliases[pkgIdent.Name]
		if !ok {
			return functionKey{}, false
		}
		candidates := byImport[importPath]
		if len(candidates) == 0 {
			return functionKey{}, false
		}
		return functionKey{pkgID: candidates[0], name: fun.Sel.Name}, true
	default:
		return functionKey{}, false
	}
}

func reverseDependers(project *model.Project, rootImportPath string) map[string]struct{} {
	allowed := map[string]struct{}{
		rootImportPath: {},
	}
	if project == nil || rootImportPath == "" {
		return allowed
	}

	reverse := make(map[string][]string)
	for _, pkg := range project.Packages {
		for _, imp := range pkg.Imports {
			reverse[imp] = append(reverse[imp], pkg.ImportPath)
		}
	}

	queue := []string{rootImportPath}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, depender := range reverse[cur] {
			if _, ok := allowed[depender]; ok {
				continue
			}
			allowed[depender] = struct{}{}
			queue = append(queue, depender)
		}
	}
	return allowed
}

func trimQuotes(v string) string {
	if len(v) < 2 {
		return ""
	}
	return v[1 : len(v)-1]
}
