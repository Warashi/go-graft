package projectload

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
)

type Loader struct {
	Dir string
}

type goListPackage struct {
	ID              string
	ImportPath      string
	Dir             string
	GoFiles         []string
	TestGoFiles     []string
	XTestGoFiles    []string
	CompiledGoFiles []string
	Imports         []string
	Error           *struct {
		Err string
	}
}

func (l Loader) Load(ctx context.Context, patterns ...string) (*model.Project, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	args := append([]string{"list", "-deps", "-test", "-json"}, patterns...)
	cmd := exec.CommandContext(ctx, "go", args...)
	if l.Dir != "" {
		cmd.Dir = l.Dir
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	decoder := json.NewDecoder(&stdout)
	rawPkgs := make([]goListPackage, 0, 128)
	for {
		var pkg goListPackage
		err := decoder.Decode(&pkg)
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode go list json: %w", err)
		}
		if pkg.Error != nil {
			return nil, fmt.Errorf("go list package error for %s: %s", pkg.ImportPath, pkg.Error.Err)
		}
		rawPkgs = append(rawPkgs, pkg)
	}

	project := &model.Project{
		ByID:         make(map[string]*model.Package, len(rawPkgs)),
		ByImportPath: make(map[string]*model.Package, len(rawPkgs)),
	}

	for _, raw := range rawPkgs {
		pkg, err := buildPackage(raw)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			continue
		}
		project.Packages = append(project.Packages, pkg)
		project.ByID[pkg.ID] = pkg
		if _, ok := project.ByImportPath[pkg.ImportPath]; !ok {
			project.ByImportPath[pkg.ImportPath] = pkg
		}
	}

	return project, nil
}

func buildPackage(raw goListPackage) (*model.Package, error) {
	if raw.Dir == "" {
		return nil, nil
	}

	srcFiles := combineUniqueFiles(raw.Dir, raw.GoFiles, raw.TestGoFiles, raw.XTestGoFiles)
	compiledFiles := combineUniqueFiles(raw.Dir, raw.CompiledGoFiles)
	if len(srcFiles) == 0 && len(compiledFiles) == 0 {
		return nil, nil
	}

	fset := token.NewFileSet()
	syntax := make([]*ast.File, 0, len(srcFiles))
	syntaxByPath := make(map[string]*ast.File, len(srcFiles))
	for _, file := range srcFiles {
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parse file %s: %w", file, err)
		}
		syntax = append(syntax, node)
		syntaxByPath[file] = node
	}

	return &model.Package{
		ID:              raw.ID,
		ImportPath:      raw.ImportPath,
		Dir:             raw.Dir,
		GoFiles:         srcFiles,
		CompiledGoFiles: compiledFiles,
		Imports:         append([]string(nil), raw.Imports...),
		Fset:            fset,
		Syntax:          syntax,
		SyntaxByPath:    syntaxByPath,
	}, nil
}

func combineUniqueFiles(dir string, groups ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, group := range groups {
		for _, file := range group {
			path := file
			if !filepath.IsAbs(path) {
				path = filepath.Join(dir, path)
			}
			path = filepath.Clean(path)
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out
}
