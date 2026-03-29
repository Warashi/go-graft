package mutationpoint

import (
	"go/ast"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/project"
)

func Collect(project *project.Project, targetTypes []reflect.Type) []model.MutationPoint {
	if project == nil || len(targetTypes) == 0 {
		return nil
	}

	targetSet := make(map[reflect.Type]struct{}, len(targetTypes))
	for _, t := range targetTypes {
		if t != nil {
			targetSet[t] = struct{}{}
		}
	}

	points := make([]model.MutationPoint, 0)
	for _, pkg := range project.Packages {
		for _, filePath := range pkg.GoFiles {
			if isTestGoFile(filePath) {
				continue
			}
			file := pkg.SyntaxByPath[filePath]
			if file == nil {
				continue
			}

			var pathStack []ast.Node
			var funcStack []*ast.FuncDecl
			compiledPath := resolveCompiledFile(pkg, filePath)
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					if len(pathStack) == 0 {
						return false
					}
					last := pathStack[len(pathStack)-1]
					pathStack = pathStack[:len(pathStack)-1]
					if _, ok := last.(*ast.FuncDecl); ok && len(funcStack) > 0 {
						funcStack = funcStack[:len(funcStack)-1]
					}
					return false
				}

				pathStack = append(pathStack, n)
				if fn, ok := n.(*ast.FuncDecl); ok {
					funcStack = append(funcStack, fn)
				}

				if _, ok := targetSet[reflect.TypeOf(n)]; !ok {
					return true
				}

				path := append([]ast.Node(nil), pathStack...)
				var enclosing *ast.FuncDecl
				if len(funcStack) > 0 {
					enclosing = funcStack[len(funcStack)-1]
				}
				pos := pkg.Fset.Position(n.Pos())
				points = append(points, model.MutationPoint{
					PkgID:          pkg.ID,
					PkgImportPath:  pkg.ImportPath,
					File:           file,
					FilePath:       filePath,
					Node:           n,
					Path:           path,
					Pos:            pos,
					EnclosingFunc:  enclosing,
					PackageName:    file.Name.Name,
					CompiledGoFile: compiledPath,
				})
				return true
			})
		}
	}
	return points
}

func resolveCompiledFile(pkg *project.Package, filePath string) string {
	base := filepath.Base(filePath)
	for _, compiled := range pkg.CompiledGoFiles {
		if filepath.Base(compiled) == base {
			return compiled
		}
	}
	return filePath
}

func isTestGoFile(path string) bool {
	return strings.HasSuffix(filepath.Base(path), "_test.go")
}
