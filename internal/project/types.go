package project

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

type Package struct {
	ID              string
	ImportPath      string
	Dir             string
	GoFiles         []string
	CompiledGoFiles []string
	Imports         []string
	Fset            *token.FileSet
	Syntax          []*ast.File
	SyntaxByPath    map[string]*ast.File
	TypesInfo       *types.Info
	Raw             *packages.Package
}

type Project struct {
	Packages     []*Package
	ByID         map[string]*Package
	ByImportPath map[string]*Package
}
