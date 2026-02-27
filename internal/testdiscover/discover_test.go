package testdiscover

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestDiscoverFindsTopLevelTestFunctions(t *testing.T) {
	const src = `package sample
import (
	t "testing"
	_ "errors"
)
func TestValid(x *t.T) {}
func Testlower(x *t.T) {}
func TestNoParam() {}
func TestWrongType(x int) {}
func BenchmarkX(x *t.T) {}
type S struct{}
func (s *S) TestMethod(x *t.T) {}
`
	filePath := filepath.Clean("/tmp/sample_test.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	project := &model.Project{
		Packages: []*model.Package{
			{
				ID:           "sample",
				ImportPath:   "example.com/sample",
				GoFiles:      []string{filePath},
				SyntaxByPath: map[string]*ast.File{},
			},
		},
	}
	project.Packages[0].SyntaxByPath = map[string]*ast.File{filePath: f}

	tests := Discover(project)
	if len(tests) != 1 {
		t.Fatalf("Discover() len = %d, want 1", len(tests))
	}
	names := []string{tests[0].Name}
	if !slices.Equal(names, []string{"TestValid"}) {
		t.Fatalf("Discover() names = %v, want [TestValid]", names)
	}
}

func TestDiscoverFindsAliasToTestingTByTypesInfo(t *testing.T) {
	const src = `package sample
import t "testing"

type TestParam = *t.T

func TestAlias(x TestParam) {}
`
	filePath := filepath.Clean("/tmp/sample_alias_test.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}
	cfg := types.Config{Importer: importer.Default()}
	if _, err := cfg.Check("example.com/sample", fset, []*ast.File{f}, info); err != nil {
		t.Fatalf("types check error = %v", err)
	}

	project := &model.Project{
		Packages: []*model.Package{
			{
				ID:           "sample",
				ImportPath:   "example.com/sample",
				GoFiles:      []string{filePath},
				SyntaxByPath: map[string]*ast.File{filePath: f},
				TypesInfo:    info,
			},
		},
	}

	tests := Discover(project)
	if len(tests) != 1 {
		t.Fatalf("Discover() len = %d, want 1", len(tests))
	}
	if tests[0].Name != "TestAlias" {
		t.Fatalf("Discover() name = %q, want TestAlias", tests[0].Name)
	}
}
