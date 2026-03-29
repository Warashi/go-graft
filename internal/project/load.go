package project

import (
	"context"
	"fmt"
	"go/ast"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Loader struct {
	Dir string
}

func (l Loader) Load(ctx context.Context, patterns ...string) (*Project, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedModule,
		Tests:   true,
		Dir:     l.Dir,
		Context: ctx,
	}
	loaded, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("go/packages load failed: %w", err)
	}
	if err := collectLoadErrors(loaded); err != nil {
		return nil, err
	}

	project := &Project{
		ByID:         make(map[string]*Package, len(loaded)),
		ByImportPath: make(map[string]*Package, len(loaded)),
	}

	for _, pkg := range loaded {
		if pkg == nil || pkg.Module == nil || !pkg.Module.Main {
			continue
		}
		converted := buildPackage(pkg)
		if converted == nil {
			continue
		}
		project.Packages = append(project.Packages, converted)
		project.ByID[converted.ID] = converted
		if _, ok := project.ByImportPath[converted.ImportPath]; !ok {
			project.ByImportPath[converted.ImportPath] = converted
		}
	}

	return project, nil
}

func buildPackage(pkg *packages.Package) *Package {
	goFiles := cleanAbsPaths(pkg.GoFiles, pkg.Dir)
	compiledGoFiles := cleanAbsPaths(pkg.CompiledGoFiles, pkg.Dir)
	if len(goFiles) == 0 && len(compiledGoFiles) == 0 {
		return nil
	}

	syntax := append([]*ast.File(nil), pkg.Syntax...)
	syntaxByPath := make(map[string]*ast.File, len(syntax))
	for _, file := range syntax {
		if file == nil || pkg.Fset == nil {
			continue
		}
		pos := pkg.Fset.Position(file.Pos())
		if pos.Filename == "" {
			continue
		}
		filePath := filepath.Clean(pos.Filename)
		if !filepath.IsAbs(filePath) && pkg.Dir != "" {
			filePath = filepath.Join(pkg.Dir, filePath)
		}
		filePath = filepath.Clean(filePath)
		syntaxByPath[filePath] = file
	}

	imports := make([]string, 0, len(pkg.Imports))
	for importPath := range pkg.Imports {
		imports = append(imports, importPath)
	}
	slices.Sort(imports)

	importPath := pkg.PkgPath
	if importPath == "" {
		importPath = pkg.ID
	}

	return &Package{
		ID:              pkg.ID,
		ImportPath:      importPath,
		Dir:             pkg.Dir,
		GoFiles:         goFiles,
		CompiledGoFiles: compiledGoFiles,
		Imports:         imports,
		Fset:            pkg.Fset,
		Syntax:          syntax,
		SyntaxByPath:    syntaxByPath,
		TypesInfo:       pkg.TypesInfo,
		Raw:             pkg,
	}
}

func cleanAbsPaths(paths []string, dir string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		cleaned := filepath.Clean(path)
		if !filepath.IsAbs(cleaned) && dir != "" {
			cleaned = filepath.Join(dir, cleaned)
		}
		cleaned = filepath.Clean(cleaned)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	slices.Sort(out)
	return out
}

func collectLoadErrors(pkgs []*packages.Package) error {
	seen := make(map[string]struct{})
	var lines []string
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		for _, loadErr := range pkg.Errors {
			msg := strings.TrimSpace(loadErr.Error())
			if msg == "" {
				continue
			}
			if _, ok := seen[msg]; ok {
				continue
			}
			seen[msg] = struct{}{}
			lines = append(lines, msg)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	slices.Sort(lines)
	return fmt.Errorf("go/packages load errors:\n%s", strings.Join(lines, "\n"))
}
