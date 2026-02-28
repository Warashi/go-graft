package graft

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"reflect"
	"runtime"
	"strings"
)

type functionSpec struct {
	pkgPath      string
	functionName string
}

// RegisterFunctionCallSwap installs a function-call swap rule using package-level functions.
func RegisterFunctionCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption) {
	if e == nil {
		panic("graft: engine must not be nil")
	}

	fromSpec := parseFunctionSpec(from, "from")
	toSpec := parseFunctionSpec(to, "to")
	if fromSpec.pkgPath != toSpec.pkgPath {
		panic("graft: from and to must be in the same package")
	}

	Register[*ast.CallExpr](e, func(c *Context, n *ast.CallExpr) (*ast.CallExpr, bool) {
		if c == nil || c.Types == nil {
			return nil, false
		}

		originalCall, ok := c.Original(n).(*ast.CallExpr)
		if !ok {
			return nil, false
		}

		fromObj, ok := calledFunctionObject(c.Types, originalCall.Fun)
		if !ok || fromObj == nil || fromObj.Pkg() == nil {
			return nil, false
		}
		if fromObj.Pkg().Path() != fromSpec.pkgPath || fromObj.Name() != fromSpec.functionName {
			return nil, false
		}

		fromSig, ok := fromObj.Type().(*types.Signature)
		if !ok {
			return nil, false
		}

		toObj := fromObj.Pkg().Scope().Lookup(toSpec.functionName)
		toFunc, ok := toObj.(*types.Func)
		if !ok || toFunc == nil {
			return nil, false
		}
		toSig, ok := toFunc.Type().(*types.Signature)
		if !ok {
			return nil, false
		}
		if !types.Identical(fromSig, toSig) {
			return nil, false
		}

		mutatedFun, ok := replaceCalledFunctionName(n.Fun, fromSpec.functionName, toSpec.functionName)
		if !ok {
			return nil, false
		}

		return &ast.CallExpr{
			Fun:      mutatedFun,
			Lparen:   n.Lparen,
			Args:     append([]ast.Expr(nil), n.Args...),
			Ellipsis: n.Ellipsis,
			Rparen:   n.Rparen,
		}, true
	}, opts...)
}

func parseFunctionSpec(fn any, argName string) functionSpec {
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		panic(fmt.Sprintf("graft: %s must be a function value", argName))
	}
	if value.IsNil() {
		panic(fmt.Sprintf("graft: %s must not be nil", argName))
	}

	runtimeFn := runtime.FuncForPC(value.Pointer())
	if runtimeFn == nil {
		panic(fmt.Sprintf("graft: %s function is unavailable", argName))
	}

	fullName := runtimeFn.Name()
	if strings.HasSuffix(fullName, "-fm") {
		panic(fmt.Sprintf("graft: %s must be a package-level function", argName))
	}

	pkgPath, functionName, ok := parsePackageFunctionName(fullName)
	if !ok {
		panic(fmt.Sprintf("graft: %s must be a package-level function", argName))
	}

	return functionSpec{
		pkgPath:      pkgPath,
		functionName: functionName,
	}
}

func parsePackageFunctionName(runtimeName string) (pkgPath string, functionName string, ok bool) {
	if runtimeName == "" {
		return "", "", false
	}

	prefix, afterSlash := path.Split(runtimeName)
	if afterSlash == "" {
		return "", "", false
	}

	pkgElem, rawName, found := strings.Cut(afterSlash, ".")
	if !found || pkgElem == "" || rawName == "" {
		return "", "", false
	}

	funcName, typeArgTail, hasTypeArgs := strings.Cut(rawName, "[")
	if hasTypeArgs {
		if typeArgTail == "" || !strings.HasSuffix(typeArgTail, "]") {
			return "", "", false
		}
		rawName = funcName
	}
	if strings.ContainsAny(rawName, ".()") {
		return "", "", false
	}

	if rawName == "" || !token.IsIdentifier(rawName) {
		return "", "", false
	}

	fullPkgPath := strings.Join([]string{prefix, pkgElem}, "")
	if fullPkgPath == "" {
		return "", "", false
	}

	return fullPkgPath, rawName, true
}

func calledFunctionObject(info *types.Info, fun ast.Expr) (*types.Func, bool) {
	if info == nil || fun == nil {
		return nil, false
	}

	switch v := fun.(type) {
	case *ast.Ident:
		fn, ok := info.ObjectOf(v).(*types.Func)
		if !ok || fn == nil {
			return nil, false
		}
		return fn, true
	case *ast.SelectorExpr:
		if v.Sel == nil {
			return nil, false
		}
		fn, ok := info.ObjectOf(v.Sel).(*types.Func)
		if !ok || fn == nil {
			return nil, false
		}
		return fn, true
	case *ast.IndexExpr:
		return calledFunctionObject(info, v.X)
	case *ast.IndexListExpr:
		return calledFunctionObject(info, v.X)
	default:
		return nil, false
	}
}

func replaceCalledFunctionName(fun ast.Expr, fromName string, toName string) (ast.Expr, bool) {
	switch v := fun.(type) {
	case *ast.Ident:
		if v.Name != fromName {
			return nil, false
		}
		return &ast.Ident{
			NamePos: v.NamePos,
			Name:    toName,
			Obj:     v.Obj,
		}, true
	case *ast.SelectorExpr:
		if v.Sel == nil || v.Sel.Name != fromName {
			return nil, false
		}
		return &ast.SelectorExpr{
			X: v.X,
			Sel: &ast.Ident{
				NamePos: v.Sel.NamePos,
				Name:    toName,
			},
		}, true
	case *ast.IndexExpr:
		replacedX, ok := replaceCalledFunctionName(v.X, fromName, toName)
		if !ok {
			return nil, false
		}
		return &ast.IndexExpr{
			X:      replacedX,
			Lbrack: v.Lbrack,
			Index:  v.Index,
			Rbrack: v.Rbrack,
		}, true
	case *ast.IndexListExpr:
		replacedX, ok := replaceCalledFunctionName(v.X, fromName, toName)
		if !ok {
			return nil, false
		}
		return &ast.IndexListExpr{
			X:       replacedX,
			Lbrack:  v.Lbrack,
			Indices: append([]ast.Expr(nil), v.Indices...),
			Rbrack:  v.Rbrack,
		}, true
	default:
		return nil, false
	}
}
