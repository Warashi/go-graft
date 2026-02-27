package astcow

import (
	"go/ast"
	"reflect"
)

func shallowCopy(n ast.Node) ast.Node {
	switch orig := n.(type) {
	case *ast.File:
		cloned := *orig
		cloned.Decls = append([]ast.Decl(nil), orig.Decls...)
		cloned.Imports = append([]*ast.ImportSpec(nil), orig.Imports...)
		cloned.Unresolved = append([]*ast.Ident(nil), orig.Unresolved...)
		cloned.Comments = append([]*ast.CommentGroup(nil), orig.Comments...)
		return &cloned
	case *ast.GenDecl:
		cloned := *orig
		cloned.Specs = append([]ast.Spec(nil), orig.Specs...)
		return &cloned
	case *ast.ValueSpec:
		cloned := *orig
		cloned.Names = append([]*ast.Ident(nil), orig.Names...)
		cloned.Values = append([]ast.Expr(nil), orig.Values...)
		return &cloned
	case *ast.AssignStmt:
		cloned := *orig
		cloned.Lhs = append([]ast.Expr(nil), orig.Lhs...)
		cloned.Rhs = append([]ast.Expr(nil), orig.Rhs...)
		return &cloned
	case *ast.ReturnStmt:
		cloned := *orig
		cloned.Results = append([]ast.Expr(nil), orig.Results...)
		return &cloned
	case *ast.ExprStmt:
		cloned := *orig
		return &cloned
	case *ast.BinaryExpr:
		cloned := *orig
		return &cloned
	case *ast.UnaryExpr:
		cloned := *orig
		return &cloned
	case *ast.CallExpr:
		cloned := *orig
		cloned.Args = append([]ast.Expr(nil), orig.Args...)
		return &cloned
	case *ast.ParenExpr:
		cloned := *orig
		return &cloned
	case *ast.SelectorExpr:
		cloned := *orig
		return &cloned
	case *ast.StarExpr:
		cloned := *orig
		return &cloned
	case *ast.FuncDecl:
		cloned := *orig
		return &cloned
	case *ast.FuncType:
		cloned := *orig
		return &cloned
	case *ast.FieldList:
		cloned := *orig
		cloned.List = append([]*ast.Field(nil), orig.List...)
		return &cloned
	case *ast.Field:
		cloned := *orig
		cloned.Names = append([]*ast.Ident(nil), orig.Names...)
		return &cloned
	case *ast.BlockStmt:
		cloned := *orig
		cloned.List = append([]ast.Stmt(nil), orig.List...)
		return &cloned
	case *ast.IfStmt:
		cloned := *orig
		return &cloned
	case *ast.ForStmt:
		cloned := *orig
		return &cloned
	case *ast.RangeStmt:
		cloned := *orig
		return &cloned
	case *ast.DeclStmt:
		cloned := *orig
		return &cloned
	case *ast.CompositeLit:
		cloned := *orig
		cloned.Elts = append([]ast.Expr(nil), orig.Elts...)
		return &cloned
	case *ast.KeyValueExpr:
		cloned := *orig
		return &cloned
	case *ast.IndexExpr:
		cloned := *orig
		return &cloned
	case *ast.IndexListExpr:
		cloned := *orig
		cloned.Indices = append([]ast.Expr(nil), orig.Indices...)
		return &cloned
	case *ast.SliceExpr:
		cloned := *orig
		return &cloned
	case *ast.SendStmt:
		cloned := *orig
		return &cloned
	case *ast.LabeledStmt:
		cloned := *orig
		return &cloned
	case *ast.SwitchStmt:
		cloned := *orig
		return &cloned
	case *ast.TypeSwitchStmt:
		cloned := *orig
		return &cloned
	case *ast.CaseClause:
		cloned := *orig
		cloned.List = append([]ast.Expr(nil), orig.List...)
		cloned.Body = append([]ast.Stmt(nil), orig.Body...)
		return &cloned
	case *ast.TypeAssertExpr:
		cloned := *orig
		return &cloned
	case *ast.IncDecStmt:
		cloned := *orig
		return &cloned
	case *ast.BranchStmt:
		cloned := *orig
		return &cloned
	case *ast.ImportSpec:
		cloned := *orig
		return &cloned
	case *ast.TypeSpec:
		cloned := *orig
		return &cloned
	case *ast.StructType:
		cloned := *orig
		return &cloned
	case *ast.InterfaceType:
		cloned := *orig
		return &cloned
	case *ast.ArrayType:
		cloned := *orig
		return &cloned
	case *ast.MapType:
		cloned := *orig
		return &cloned
	case *ast.ChanType:
		cloned := *orig
		return &cloned
	case *ast.FuncLit:
		cloned := *orig
		return &cloned
	default:
		return nil
	}
}

func replaceChild(parent ast.Node, oldChild ast.Node, newChild ast.Node) bool {
	switch p := parent.(type) {
	case *ast.File:
		if replaced, ok := replaceSlice(p.Decls, oldChild, newChild); ok {
			p.Decls = replaced
			return true
		}
		if replaced, ok := replaceSlice(p.Imports, oldChild, newChild); ok {
			p.Imports = replaced
			return true
		}
		if replaced, ok := replaceSlice(p.Comments, oldChild, newChild); ok {
			p.Comments = replaced
			return true
		}
		return false
	case *ast.GenDecl:
		replaced, ok := replaceSlice(p.Specs, oldChild, newChild)
		if !ok {
			return false
		}
		p.Specs = replaced
		return true
	case *ast.ValueSpec:
		if replaceField(&p.Type, oldChild, newChild) {
			return true
		}
		if replaced, ok := replaceSlice(p.Names, oldChild, newChild); ok {
			p.Names = replaced
			return true
		}
		if replaced, ok := replaceSlice(p.Values, oldChild, newChild); ok {
			p.Values = replaced
			return true
		}
		return false
	case *ast.AssignStmt:
		if replaced, ok := replaceSlice(p.Lhs, oldChild, newChild); ok {
			p.Lhs = replaced
			return true
		}
		if replaced, ok := replaceSlice(p.Rhs, oldChild, newChild); ok {
			p.Rhs = replaced
			return true
		}
		return false
	case *ast.ReturnStmt:
		replaced, ok := replaceSlice(p.Results, oldChild, newChild)
		if !ok {
			return false
		}
		p.Results = replaced
		return true
	case *ast.ExprStmt:
		return replaceField(&p.X, oldChild, newChild)
	case *ast.BinaryExpr:
		return replaceField(&p.X, oldChild, newChild) || replaceField(&p.Y, oldChild, newChild)
	case *ast.UnaryExpr:
		return replaceField(&p.X, oldChild, newChild)
	case *ast.CallExpr:
		if replaceField(&p.Fun, oldChild, newChild) {
			return true
		}
		replaced, ok := replaceSlice(p.Args, oldChild, newChild)
		if !ok {
			return false
		}
		p.Args = replaced
		return true
	case *ast.ParenExpr:
		return replaceField(&p.X, oldChild, newChild)
	case *ast.SelectorExpr:
		return replaceField(&p.X, oldChild, newChild) || replaceField(&p.Sel, oldChild, newChild)
	case *ast.StarExpr:
		return replaceField(&p.X, oldChild, newChild)
	case *ast.FuncDecl:
		if replaceField(&p.Name, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Recv, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Type, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Body, oldChild, newChild)
	case *ast.FuncType:
		if replaceField(&p.TypeParams, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Params, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Results, oldChild, newChild)
	case *ast.FieldList:
		replaced, ok := replaceSlice(p.List, oldChild, newChild)
		if !ok {
			return false
		}
		p.List = replaced
		return true
	case *ast.Field:
		if replaced, ok := replaceSlice(p.Names, oldChild, newChild); ok {
			p.Names = replaced
			return true
		}
		if replaceField(&p.Type, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Tag, oldChild, newChild)
	case *ast.BlockStmt:
		replaced, ok := replaceSlice(p.List, oldChild, newChild)
		if !ok {
			return false
		}
		p.List = replaced
		return true
	case *ast.IfStmt:
		if replaceField(&p.Init, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Cond, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Body, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Else, oldChild, newChild)
	case *ast.ForStmt:
		if replaceField(&p.Init, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Cond, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Post, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Body, oldChild, newChild)
	case *ast.RangeStmt:
		if replaceField(&p.Key, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Value, oldChild, newChild) {
			return true
		}
		if replaceField(&p.X, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Body, oldChild, newChild)
	case *ast.DeclStmt:
		return replaceField(&p.Decl, oldChild, newChild)
	case *ast.CompositeLit:
		if replaceField(&p.Type, oldChild, newChild) {
			return true
		}
		replaced, ok := replaceSlice(p.Elts, oldChild, newChild)
		if !ok {
			return false
		}
		p.Elts = replaced
		return true
	case *ast.KeyValueExpr:
		return replaceField(&p.Key, oldChild, newChild) || replaceField(&p.Value, oldChild, newChild)
	case *ast.IndexExpr:
		return replaceField(&p.X, oldChild, newChild) || replaceField(&p.Index, oldChild, newChild)
	case *ast.IndexListExpr:
		if replaceField(&p.X, oldChild, newChild) {
			return true
		}
		replaced, ok := replaceSlice(p.Indices, oldChild, newChild)
		if !ok {
			return false
		}
		p.Indices = replaced
		return true
	case *ast.SliceExpr:
		if replaceField(&p.X, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Low, oldChild, newChild) {
			return true
		}
		if replaceField(&p.High, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Max, oldChild, newChild)
	case *ast.SendStmt:
		return replaceField(&p.Chan, oldChild, newChild) || replaceField(&p.Value, oldChild, newChild)
	case *ast.LabeledStmt:
		return replaceField(&p.Label, oldChild, newChild) || replaceField(&p.Stmt, oldChild, newChild)
	case *ast.SwitchStmt:
		if replaceField(&p.Init, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Tag, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Body, oldChild, newChild)
	case *ast.TypeSwitchStmt:
		if replaceField(&p.Init, oldChild, newChild) {
			return true
		}
		if replaceField(&p.Assign, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Body, oldChild, newChild)
	case *ast.CaseClause:
		if replaced, ok := replaceSlice(p.List, oldChild, newChild); ok {
			p.List = replaced
			return true
		}
		if replaced, ok := replaceSlice(p.Body, oldChild, newChild); ok {
			p.Body = replaced
			return true
		}
		return false
	case *ast.TypeAssertExpr:
		return replaceField(&p.X, oldChild, newChild) || replaceField(&p.Type, oldChild, newChild)
	case *ast.IncDecStmt:
		return replaceField(&p.X, oldChild, newChild)
	case *ast.BranchStmt:
		return replaceField(&p.Label, oldChild, newChild)
	case *ast.ImportSpec:
		if replaceField(&p.Name, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Path, oldChild, newChild)
	case *ast.TypeSpec:
		if replaceField(&p.Name, oldChild, newChild) {
			return true
		}
		if replaceField(&p.TypeParams, oldChild, newChild) {
			return true
		}
		return replaceField(&p.Type, oldChild, newChild)
	case *ast.StructType:
		return replaceField(&p.Fields, oldChild, newChild)
	case *ast.InterfaceType:
		return replaceField(&p.Methods, oldChild, newChild)
	case *ast.ArrayType:
		return replaceField(&p.Len, oldChild, newChild) || replaceField(&p.Elt, oldChild, newChild)
	case *ast.MapType:
		return replaceField(&p.Key, oldChild, newChild) || replaceField(&p.Value, oldChild, newChild)
	case *ast.ChanType:
		return replaceField(&p.Value, oldChild, newChild)
	case *ast.FuncLit:
		return replaceField(&p.Type, oldChild, newChild) || replaceField(&p.Body, oldChild, newChild)
	case *ast.GoStmt:
		return replaceField(&p.Call, oldChild, newChild)
	case *ast.DeferStmt:
		return replaceField(&p.Call, oldChild, newChild)
	default:
		return false
	}
}

func replaceField[T ast.Node](field *T, oldChild ast.Node, newChild ast.Node) bool {
	if field == nil {
		return false
	}
	current := ast.Node(*field)
	if !sameNode(current, oldChild) {
		return false
	}
	replaced, ok := newChild.(T)
	if !ok {
		return false
	}
	*field = replaced
	return true
}

func replaceSlice[T ast.Node](items []T, oldChild ast.Node, newChild ast.Node) ([]T, bool) {
	for i := range items {
		current := ast.Node(items[i])
		if !sameNode(current, oldChild) {
			continue
		}
		replaced, ok := newChild.(T)
		if !ok {
			return nil, false
		}
		out := append([]T(nil), items...)
		out[i] = replaced
		return out, true
	}
	return nil, false
}

func sameNode(a ast.Node, b ast.Node) bool {
	if a == nil || b == nil {
		return a == b
	}
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if va.Kind() != reflect.Pointer || vb.Kind() != reflect.Pointer {
		return false
	}
	return va.Type() == vb.Type() && va.Pointer() == vb.Pointer()
}
