package model

import (
	"go/ast"
	"go/token"
	"go/types"
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
}

type Project struct {
	Packages     []*Package
	ByID         map[string]*Package
	ByImportPath map[string]*Package
}

type TestRef struct {
	PkgID       string
	ImportPath  string
	Name        string
	FilePath    string
	PackageName string
}

type MutationPoint struct {
	PkgID          string
	PkgImportPath  string
	File           *ast.File
	FilePath       string
	Node           ast.Node
	Path           []ast.Node
	Pos            token.Position
	EnclosingFunc  *ast.FuncDecl
	PackageName    string
	CompiledGoFile string
}

type Mutant struct {
	ID            string
	RuleName      string
	Point         MutationPoint
	WorkDir       string
	TempDir       string
	OverlayPath   string
	OverlayTmpDir string
	MutantFile    string
	ReplaceMap    map[string]string
	SelectedTests SelectedTests
}

type SelectedTests struct {
	ByImportPath map[string][]string
}

type MutantExecResult struct {
	Mutant         Mutant
	Status         MutantStatus
	Reason         string
	ExecutedPkgs   map[string]string
	FailedCommand  []string
	Stdout         string
	Stderr         string
	TimedOut       bool
	ElapsedNsec    int64
	InternalErrMsg string
}

type MutantStatus int

const (
	MutantKilled MutantStatus = iota
	MutantSurvived
	MutantUnsupported
	MutantErrored
)
