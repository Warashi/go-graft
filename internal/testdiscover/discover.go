package testdiscover

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Warashi/go-graft/internal/callresolve"
	"github.com/Warashi/go-graft/internal/model"
)

const (
	ExcludeReasonAutoRunReachable = "auto-run-reachable"
	ExcludeReasonDirectiveExclude = "directive-exclude"
)

type ExcludedTest struct {
	Ref    model.TestRef
	Reason string
}

type Result struct {
	Included []model.TestRef
	Excluded []ExcludedTest
}

type functionKey struct {
	pkgID string
	name  string
}

type directive int

const (
	directiveNone directive = iota
	directiveInclude
	directiveExclude
)

type functionInfo struct {
	calls        []functionKey
	hasEngineRun bool
	directive    directive
}

type functionRecord struct {
	key           functionKey
	typesInfo     *types.Info
	importAliases map[string]string
	body          *ast.BlockStmt
}

func Discover(project *model.Project) []model.TestRef {
	return DiscoverDetailed(project).Included
}

func DiscoverDetailed(project *model.Project) Result {
	if project == nil {
		return Result{}
	}
	out := Result{}
	tests := make([]model.TestRef, 0)
	infos, records := collectFunctions(project)

	for _, pkg := range project.Packages {
		for _, filePath := range pkg.GoFiles {
			file := pkg.SyntaxByPath[filePath]
			if file == nil {
				continue
			}
			testingAliases := testingImportAliases(file)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok {
					continue
				}
				if !isTopLevelTest(fn, testingAliases, pkg.TypesInfo) {
					continue
				}
				tests = append(tests, model.TestRef{
					PkgID:       pkg.ID,
					ImportPath:  pkg.ImportPath,
					Name:        fn.Name.Name,
					FilePath:    filepath.Clean(filePath),
					PackageName: file.Name.Name,
				})
			}
		}
	}

	buildCalls(project, infos, records)
	out.Included = make([]model.TestRef, 0, len(tests))
	out.Excluded = make([]ExcludedTest, 0)
	for _, test := range tests {
		key := functionKey{pkgID: test.PkgID, name: test.Name}
		info := infos[key]
		switch resolvedDirective(info) {
		case directiveInclude:
			out.Included = append(out.Included, test)
		case directiveExclude:
			out.Excluded = append(out.Excluded, ExcludedTest{
				Ref:    test,
				Reason: ExcludeReasonDirectiveExclude,
			})
		default:
			if reachesEngineRun(key, infos) {
				out.Excluded = append(out.Excluded, ExcludedTest{
					Ref:    test,
					Reason: ExcludeReasonAutoRunReachable,
				})
				continue
			}
			out.Included = append(out.Included, test)
		}
	}
	return out
}

func collectFunctions(project *model.Project) (map[functionKey]*functionInfo, []functionRecord) {
	if project == nil {
		return nil, nil
	}
	infos := make(map[functionKey]*functionInfo)
	records := make([]functionRecord, 0)
	for _, pkg := range project.Packages {
		if pkg == nil {
			continue
		}
		for _, filePath := range pkg.GoFiles {
			file := pkg.SyntaxByPath[filePath]
			if file == nil {
				continue
			}
				aliases := callresolve.ImportAliases(file)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Body == nil {
					continue
				}
				key := functionKey{pkgID: pkg.ID, name: fn.Name.Name}
				if _, ok := infos[key]; !ok {
					infos[key] = &functionInfo{}
				}
				infos[key].directive = parseDirective(fn)
				records = append(records, functionRecord{
					key:           key,
					typesInfo:     pkg.TypesInfo,
					importAliases: aliases,
					body:          fn.Body,
				})
			}
		}
	}
	return infos, records
}

func buildCalls(project *model.Project, infos map[functionKey]*functionInfo, records []functionRecord) {
	if len(infos) == 0 || len(records) == 0 {
		return
	}
	byImport := callresolve.BuildByImport(project)

	for _, rec := range records {
		info := infos[rec.key]
		if info == nil {
			continue
		}
		seenCalls := make(map[functionKey]struct{})
		ast.Inspect(rec.body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			if isGoGraftEngineRunCall(call, rec.typesInfo) {
				info.hasEngineRun = true
			}
			if callee, ok := resolveCallTarget(rec.key.pkgID, call, rec.typesInfo, rec.importAliases, byImport, infos); ok {
				if _, ok := seenCalls[callee]; !ok {
					seenCalls[callee] = struct{}{}
					info.calls = append(info.calls, callee)
				}
			}
			return true
		})
	}
}

func resolvedDirective(info *functionInfo) directive {
	if info == nil {
		return directiveNone
	}
	return info.directive
}

func reachesEngineRun(start functionKey, infos map[functionKey]*functionInfo) bool {
	seen := map[functionKey]struct{}{}
	var visit func(functionKey) bool
	visit = func(key functionKey) bool {
		if _, ok := seen[key]; ok {
			return false
		}
		seen[key] = struct{}{}

		info, ok := infos[key]
		if !ok || info == nil {
			return false
		}
		if info.hasEngineRun {
			return true
		}
		return slices.ContainsFunc(info.calls, visit)
	}
	return visit(start)
}

func parseDirective(fn *ast.FuncDecl) directive {
	if fn == nil {
		return directiveNone
	}
	hasInclude := hasDirective(fn.Doc, "gograft:include")
	hasExclude := hasDirective(fn.Doc, "gograft:exclude")
	if hasInclude {
		return directiveInclude
	}
	if hasExclude {
		return directiveExclude
	}
	return directiveNone
}

func hasDirective(group *ast.CommentGroup, token string) bool {
	if group == nil || token == "" {
		return false
	}
	for _, c := range group.List {
		if c == nil {
			continue
		}
		normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Text), " ", ""))
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func resolveCallTarget(currentPkgID string, call *ast.CallExpr, info *types.Info, aliases map[string]string, byImport map[string][]string, infos map[functionKey]*functionInfo) (functionKey, bool) {
	resolved, ok := callresolve.ResolveFunctionCall(currentPkgID, call, info, aliases, byImport, func(key callresolve.FunctionKey) bool {
		_, ok := infos[functionKey{pkgID: key.PkgID, name: key.Name}]
		return ok
	})
	if !ok {
		return functionKey{}, false
	}
	return functionKey{pkgID: resolved.PkgID, name: resolved.Name}, true
}

func isGoGraftEngineRunCall(call *ast.CallExpr, info *types.Info) bool {
	if info == nil || call == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Run" {
		return false
	}
	fn, ok := info.Uses[sel.Sel].(*types.Func)
	if !ok || fn == nil || fn.Name() != "Run" {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() == nil {
		return false
	}
	recv := types.Unalias(sig.Recv().Type())
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = types.Unalias(ptr.Elem())
	}
	named, ok := recv.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "github.com/Warashi/go-graft" && named.Obj().Name() == "Engine"
}

func isTopLevelTest(fn *ast.FuncDecl, testingAliases map[string]struct{}, typesInfo *types.Info) bool {
	if fn == nil || fn.Name == nil || fn.Recv != nil || fn.Type == nil || fn.Type.Params == nil {
		return false
	}
	if !looksLikeTestName(fn.Name.Name) {
		return false
	}
	if len(fn.Type.Params.List) != 1 {
		return false
	}
	if len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}
	return isPointerToTestingT(fn.Type.Params.List[0].Type, testingAliases, typesInfo)
}

func looksLikeTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	rest := strings.TrimPrefix(name, "Test")
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return !unicode.IsLower(r)
}

func isPointerToTestingT(expr ast.Expr, testingAliases map[string]struct{}, typesInfo *types.Info) bool {
	if isPointerToTestingTByTypes(expr, typesInfo) {
		return true
	}

	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if sel.Sel == nil || sel.Sel.Name != "T" {
		return false
	}
	_, ok = testingAliases[pkgIdent.Name]
	return ok
}

func isPointerToTestingTByTypes(expr ast.Expr, info *types.Info) bool {
	if info == nil || expr == nil {
		return false
	}
	typ := types.Unalias(info.TypeOf(expr))
	ptr, ok := typ.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == "testing" && obj.Name() == "T"
}

func testingImportAliases(file *ast.File) map[string]struct{} {
	out := map[string]struct{}{
		"testing": {},
	}
	if file == nil {
		return out
	}
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		path := strings.Trim(imp.Path.Value, "\"")
		if path != "testing" {
			continue
		}
		if imp.Name == nil {
			out["testing"] = struct{}{}
			continue
		}
		if imp.Name.Name == "." || imp.Name.Name == "_" {
			continue
		}
		out[imp.Name.Name] = struct{}{}
	}
	return out
}
