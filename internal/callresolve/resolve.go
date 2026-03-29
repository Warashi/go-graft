package callresolve

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Warashi/go-graft/internal/project"
)

// FunctionKey identifies one top-level function in one package.
type FunctionKey struct {
	PkgID string
	Name  string
}

// ExistsFunc checks whether the resolved key is considered valid by caller.
type ExistsFunc func(key FunctionKey) bool

// ImportAliases builds alias -> importPath map from one file.
func ImportAliases(file *ast.File) map[string]string {
	out := make(map[string]string)
	if file == nil {
		return out
	}
	for _, imp := range file.Imports {
		if imp.Path == nil {
			continue
		}
		importPath := strings.Trim(imp.Path.Value, "\"")
		if importPath == "" {
			continue
		}
		alias := filepath.Base(importPath)
		if imp.Name != nil {
			switch imp.Name.Name {
			case ".", "_":
				continue
			default:
				alias = imp.Name.Name
			}
		}
		out[alias] = importPath
	}
	return out
}

// BuildByImport builds importPath -> sorted package IDs.
func BuildByImport(project *project.Project) map[string][]string {
	byImport := make(map[string][]string)
	if project == nil {
		return byImport
	}
	for _, pkg := range project.Packages {
		if pkg == nil || pkg.ImportPath == "" || pkg.ID == "" {
			continue
		}
		byImport[pkg.ImportPath] = append(byImport[pkg.ImportPath], pkg.ID)
	}
	for importPath := range byImport {
		ids := append([]string(nil), byImport[importPath]...)
		slices.Sort(ids)
		ids = slices.Compact(ids)
		byImport[importPath] = ids
	}
	return byImport
}

// ResolveFunctionCall resolves top-level function call target.
// It first tries type info and falls back to syntax-level resolution.
func ResolveFunctionCall(currentPkgID string, call *ast.CallExpr, info *types.Info, aliases map[string]string, byImport map[string][]string, exists ExistsFunc) (FunctionKey, bool) {
	if key, ok := resolveByTypes(currentPkgID, call, info, byImport, exists); ok {
		return key, true
	}
	return resolveBySyntax(currentPkgID, call, aliases, byImport, exists)
}

func resolveByTypes(currentPkgID string, call *ast.CallExpr, info *types.Info, byImport map[string][]string, exists ExistsFunc) (FunctionKey, bool) {
	if info == nil || call == nil {
		return FunctionKey{}, false
	}

	resolveObj := func(obj types.Object) (FunctionKey, bool) {
		fn, ok := obj.(*types.Func)
		if !ok || fn == nil || fn.Pkg() == nil {
			return FunctionKey{}, false
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok {
			return FunctionKey{}, false
		}
		if sig.Recv() != nil {
			return FunctionKey{}, false
		}
		return lookupFunctionKey(fn.Pkg().Path(), fn.Name(), currentPkgID, byImport, exists)
	}

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return resolveObj(info.Uses[fun])
	case *ast.SelectorExpr:
		return resolveObj(info.Uses[fun.Sel])
	default:
		return FunctionKey{}, false
	}
}

func resolveBySyntax(currentPkgID string, call *ast.CallExpr, aliases map[string]string, byImport map[string][]string, exists ExistsFunc) (FunctionKey, bool) {
	if call == nil {
		return FunctionKey{}, false
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		key := FunctionKey{PkgID: currentPkgID, Name: fun.Name}
		return key, keyExists(key, exists)
	case *ast.SelectorExpr:
		pkgIdent, ok := fun.X.(*ast.Ident)
		if !ok {
			return FunctionKey{}, false
		}
		importPath, ok := aliases[pkgIdent.Name]
		if !ok {
			return FunctionKey{}, false
		}
		return lookupFunctionKey(importPath, fun.Sel.Name, currentPkgID, byImport, exists)
	default:
		return FunctionKey{}, false
	}
}

func lookupFunctionKey(importPath string, name string, currentPkgID string, byImport map[string][]string, exists ExistsFunc) (FunctionKey, bool) {
	candidates := append([]string(nil), byImport[importPath]...)
	if len(candidates) == 0 {
		return FunctionKey{}, false
	}
	slices.Sort(candidates)
	candidates = slices.Compact(candidates)

	for _, pkgID := range candidates {
		if pkgID != currentPkgID {
			continue
		}
		key := FunctionKey{PkgID: pkgID, Name: name}
		if keyExists(key, exists) {
			return key, true
		}
	}
	for _, pkgID := range candidates {
		key := FunctionKey{PkgID: pkgID, Name: name}
		if keyExists(key, exists) {
			return key, true
		}
	}
	return FunctionKey{}, false
}

func keyExists(key FunctionKey, exists ExistsFunc) bool {
	if exists == nil {
		return true
	}
	return exists(key)
}
