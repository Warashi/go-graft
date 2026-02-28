package graft

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Context is passed to each rule callback.
type Context struct {
	Fset  *token.FileSet
	Pkg   *packages.Package
	File  *ast.File
	Types *types.Info
	Path  []ast.Node

	cloneMap map[ast.Node]ast.Node
}

func newContext() *Context {
	return &Context{
		cloneMap: make(map[ast.Node]ast.Node),
	}
}

func (c *Context) setOriginal(clone ast.Node, original ast.Node) {
	if c == nil || clone == nil || original == nil {
		return
	}
	if c.cloneMap == nil {
		c.cloneMap = make(map[ast.Node]ast.Node)
	}
	c.cloneMap[clone] = original
}

// Original returns the original node from which clone was copied.
func (c *Context) Original(node ast.Node) ast.Node {
	if c == nil || node == nil {
		return nil
	}
	current := node
	seen := map[ast.Node]struct{}{}
	for {
		if _, ok := seen[current]; ok {
			return current
		}
		seen[current] = struct{}{}

		original, ok := c.cloneMap[current]
		if !ok {
			return current
		}
		current = original
	}
}

// TypeOf returns static type of the node if available.
func (c *Context) TypeOf(node ast.Node) types.Type {
	if c == nil || c.Types == nil || node == nil {
		return nil
	}
	original := c.Original(node)
	if expr, ok := original.(ast.Expr); ok {
		return c.Types.TypeOf(expr)
	}
	if ident, ok := original.(*ast.Ident); ok {
		obj := c.Types.ObjectOf(ident)
		if obj == nil {
			return nil
		}
		return obj.Type()
	}
	return nil
}
