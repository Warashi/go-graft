package rule

import (
	"go/ast"
	"go/types"
	"reflect"
	"runtime"
	"strings"
)

type methodExprSpec struct {
	receiverPkgPath string
	receiverType    string
	methodName      string
}

// RegisterMethodCallSwap installs a method-call swap rule using method expressions.
func RegisterMethodCallSwap[F any](registry *Registry, from F, to F, opts ...Option) {
	if registry == nil {
		panic("graft: registry must not be nil")
	}

	fromSpec := parseMethodExprSpec(from, "from")
	toSpec := parseMethodExprSpec(to, "to")

	if fromSpec.receiverPkgPath != toSpec.receiverPkgPath || fromSpec.receiverType != toSpec.receiverType {
		panic("graft: from and to must have the same receiver type")
	}

	Register[*ast.CallExpr](registry, func(c *Context, n *ast.CallExpr) (*ast.CallExpr, bool) {
		if c == nil || c.Types == nil {
			return nil, false
		}

		sel, ok := n.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != fromSpec.methodName {
			return nil, false
		}

		originalCall, ok := c.Original(n).(*ast.CallExpr)
		if !ok {
			return nil, false
		}
		originalSel, ok := originalCall.Fun.(*ast.SelectorExpr)
		if !ok || originalSel.Sel == nil || originalSel.Sel.Name != fromSpec.methodName {
			return nil, false
		}

		fromObj, ok := c.Types.ObjectOf(originalSel.Sel).(*types.Func)
		if !ok || fromObj == nil {
			return nil, false
		}
		fromSig, ok := fromObj.Type().(*types.Signature)
		if !ok || fromSig.Recv() == nil {
			return nil, false
		}

		recvType := types.Unalias(fromSig.Recv().Type())
		if !matchesReceiverType(recvType, fromSpec.receiverPkgPath, fromSpec.receiverType) {
			return nil, false
		}

		toObj, _, _ := types.LookupFieldOrMethod(recvType, false, nil, toSpec.methodName)
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

		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: sel.X,
				Sel: &ast.Ident{
					NamePos: sel.Sel.NamePos,
					Name:    toSpec.methodName,
				},
			},
			Lparen:   n.Lparen,
			Args:     append([]ast.Expr(nil), n.Args...),
			Ellipsis: n.Ellipsis,
			Rparen:   n.Rparen,
		}, true
	}, opts...)
}

func parseMethodExprSpec(fn any, argName string) methodExprSpec {
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		panic("graft: " + argName + " must be a function value")
	}
	if value.IsNil() {
		panic("graft: " + argName + " must not be nil")
	}

	fnType := value.Type()
	if fnType.NumIn() == 0 {
		panic("graft: " + argName + " must be a method expression")
	}

	recvPkgPath, recvTypeName, ok := parseReceiverFromFuncType(fnType.In(0))
	if !ok {
		panic("graft: " + argName + " must have a named receiver type")
	}

	runtimeFn := runtime.FuncForPC(value.Pointer())
	if runtimeFn == nil {
		panic("graft: " + argName + " method function is unavailable")
	}

	fullName := runtimeFn.Name()
	if strings.HasSuffix(fullName, "-fm") {
		panic("graft: " + argName + " must be a method expression, not a method value")
	}

	methodName, ok := parseMethodName(fullName, recvPkgPath, recvTypeName)
	if !ok {
		panic("graft: " + argName + " must be a method expression")
	}

	return methodExprSpec{
		receiverPkgPath: recvPkgPath,
		receiverType:    recvTypeName,
		methodName:      methodName,
	}
}

func parseReceiverFromFuncType(recv reflect.Type) (pkgPath string, typeName string, ok bool) {
	if recv.Kind() == reflect.Pointer {
		recv = recv.Elem()
	}
	if recv.Name() == "" || recv.PkgPath() == "" {
		return "", "", false
	}
	return recv.PkgPath(), recv.Name(), true
}

func parseMethodName(runtimeName string, recvPkgPath string, recvTypeName string) (string, bool) {
	ptrPrefix := recvPkgPath + ".(*" + recvTypeName + ")."
	if method, ok := strings.CutPrefix(runtimeName, ptrPrefix); ok {
		if method == "" || strings.Contains(method, ".") {
			return "", false
		}
		return method, true
	}

	valuePrefix := recvPkgPath + "." + recvTypeName + "."
	if method, ok := strings.CutPrefix(runtimeName, valuePrefix); ok {
		if method == "" || strings.Contains(method, ".") {
			return "", false
		}
		return method, true
	}

	return "", false
}

func matchesReceiverType(typ types.Type, wantPkgPath string, wantTypeName string) bool {
	current := types.Unalias(typ)
	if ptr, ok := current.(*types.Pointer); ok {
		current = types.Unalias(ptr.Elem())
	}
	named, ok := current.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == wantPkgPath && obj.Name() == wantTypeName
}
