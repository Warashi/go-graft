package selection

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
	"github.com/Warashi/go-graft/internal/project"
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
	project := parseOneFileProject(t, "sample", "example.com/sample", "/tmp/sample_test.go", src, nil)

	result := DiscoverDetailed(project)
	if len(result.Included) != 1 {
		t.Fatalf("Discover() included len = %d, want 1", len(result.Included))
	}
	if len(result.Excluded) != 0 {
		t.Fatalf("Discover() excluded len = %d, want 0", len(result.Excluded))
	}
	names := []string{result.Included[0].Name}
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

	project := &project.Project{
		Packages: []*project.Package{
			{
				ID:           "sample",
				ImportPath:   "example.com/sample",
				GoFiles:      []string{filePath},
				SyntaxByPath: map[string]*ast.File{filePath: f},
				TypesInfo:    info,
			},
		},
	}

	result := DiscoverDetailed(project)
	if len(result.Included) != 1 {
		t.Fatalf("Discover() included len = %d, want 1", len(result.Included))
	}
	if len(result.Excluded) != 0 {
		t.Fatalf("Discover() excluded len = %d, want 0", len(result.Excluded))
	}
	if result.Included[0].Name != "TestAlias" {
		t.Fatalf("Discover() name = %q, want TestAlias", result.Included[0].Name)
	}
}

func TestDiscoverExcludesTestReachableToEngineRunRecursively(t *testing.T) {
	const src = `package sample
import "testing"

func helper1(e any) { helper2(e) }
func helper2(e any) { e.Run() }

func TestMutation(t *testing.T) { helper1(nil) }
func TestRegular(t *testing.T) {}
`
	filePath := filepath.Clean("/tmp/sample_mutation_test.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	project := parseOneFileProject(t, "sample", "example.com/sample", filePath, src, nil)
	project.Packages[0].SyntaxByPath[filePath] = file
	project.Packages[0].TypesInfo = withSyntheticEngineRun(file)

	result := DiscoverDetailed(project)
	if !hasIncluded(result.Included, "TestRegular") {
		t.Fatalf("Discover() included = %v, want TestRegular", names(result.Included))
	}
	if hasIncluded(result.Included, "TestMutation") {
		t.Fatalf("Discover() included = %v, TestMutation should be excluded", names(result.Included))
	}
	if !hasExcluded(result.Excluded, "TestMutation", ExcludeReasonAutoRunReachable) {
		t.Fatalf("Discover() excluded = %v, want TestMutation(%s)", result.Excluded, ExcludeReasonAutoRunReachable)
	}
}

func TestDiscoverDirectivesOverrideAutoExclusion(t *testing.T) {
	const src = `package sample
import "testing"

func helper(e any) { e.Run() }

//gograft:include
func TestKeep(t *testing.T) { helper(nil) }

//gograft:exclude
func TestDrop(t *testing.T) {}

//gograft:exclude
//gograft:include
func TestIncludeWins(t *testing.T) { helper(nil) }
`
	filePath := filepath.Clean("/tmp/sample_directive_test.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	project := parseOneFileProject(t, "sample", "example.com/sample", filePath, src, nil)
	project.Packages[0].SyntaxByPath[filePath] = file
	project.Packages[0].TypesInfo = withSyntheticEngineRun(file)

	result := DiscoverDetailed(project)
	if !hasIncluded(result.Included, "TestKeep") {
		t.Fatalf("Discover() included = %v, want TestKeep", names(result.Included))
	}
	if !hasIncluded(result.Included, "TestIncludeWins") {
		t.Fatalf("Discover() included = %v, want TestIncludeWins", names(result.Included))
	}
	if hasIncluded(result.Included, "TestDrop") {
		t.Fatalf("Discover() included = %v, TestDrop should be excluded", names(result.Included))
	}
	if !hasExcluded(result.Excluded, "TestDrop", ExcludeReasonDirectiveExclude) {
		t.Fatalf("Discover() excluded = %v, want TestDrop(%s)", result.Excluded, ExcludeReasonDirectiveExclude)
	}
}

func parseOneFileProject(t *testing.T, pkgID string, importPath string, filePath string, src string, info *types.Info) *project.Project {
	t.Helper()
	filePath = filepath.Clean(filePath)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	return &project.Project{
		Packages: []*project.Package{
			{
				ID:           pkgID,
				ImportPath:   importPath,
				GoFiles:      []string{filePath},
				SyntaxByPath: map[string]*ast.File{filePath: file},
				TypesInfo:    info,
			},
		},
	}
}

func withSyntheticEngineRun(file *ast.File) *types.Info {
	info := &types.Info{
		Uses: make(map[*ast.Ident]types.Object),
	}
	runObj := syntheticGoGraftEngineRunMethod()
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Run" {
			return true
		}
		info.Uses[sel.Sel] = runObj
		return true
	})
	return info
}

func syntheticGoGraftEngineRunMethod() *types.Func {
	pkg := types.NewPackage("github.com/Warashi/go-graft", "graft")
	engineObj := types.NewTypeName(token.NoPos, pkg, "Engine", nil)
	engineNamed := types.NewNamed(engineObj, types.NewStruct(nil, nil), nil)
	recv := types.NewVar(token.NoPos, pkg, "e", types.NewPointer(engineNamed))
	sig := types.NewSignatureType(recv, nil, nil, nil, nil, false)
	return types.NewFunc(token.NoPos, pkg, "Run", sig)
}

func hasIncluded(tests []model.TestRef, name string) bool {
	for _, test := range tests {
		if test.Name == name {
			return true
		}
	}
	return false
}

func hasExcluded(tests []ExcludedTest, name string, reason string) bool {
	for _, test := range tests {
		if test.Ref.Name == name && test.Reason == reason {
			return true
		}
	}
	return false
}

func names(tests []model.TestRef) []string {
	out := make([]string, 0, len(tests))
	for _, test := range tests {
		out = append(out, test.Name)
	}
	slices.Sort(out)
	return out
}
