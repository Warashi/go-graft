package mutationpoint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Warashi/go-graft/internal/model"
)

func TestCollectTracksPathAndEnclosingFunction(t *testing.T) {
	const src = `package p
var top = 1 + 2
func Foo() int {
	return 3 + 4
}
`
	filePath := filepath.Clean("/tmp/p.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	project := &model.Project{
		Packages: []*model.Package{
			{
				ID:              "p",
				ImportPath:      "example.com/p",
				GoFiles:         []string{filePath},
				CompiledGoFiles: []string{filePath},
				Fset:            fset,
				SyntaxByPath:    map[string]*ast.File{filePath: f},
			},
		},
	}

	points := Collect(project, []reflect.Type{reflect.TypeFor[*ast.BinaryExpr]()})
	if len(points) != 2 {
		t.Fatalf("Collect() points = %d, want 2", len(points))
	}

	var foundTop, foundFoo bool
	for _, point := range points {
		if len(point.Path) == 0 {
			t.Fatal("point.Path should not be empty")
		}
		if _, ok := point.Path[0].(*ast.File); !ok {
			t.Fatalf("path[0] type = %T, want *ast.File", point.Path[0])
		}
		if _, ok := point.Path[len(point.Path)-1].(*ast.BinaryExpr); !ok {
			t.Fatalf("path[last] type = %T, want *ast.BinaryExpr", point.Path[len(point.Path)-1])
		}
		if point.Pos.Line == 0 {
			t.Fatal("point position should include line info")
		}
		switch point.Pos.Line {
		case 2:
			foundTop = true
			if point.EnclosingFunc != nil {
				t.Fatalf("top-level point enclosing func = %v, want nil", point.EnclosingFunc.Name)
			}
		case 4:
			foundFoo = true
			if point.EnclosingFunc == nil || point.EnclosingFunc.Name.Name != "Foo" {
				t.Fatalf("function point enclosing func = %v, want Foo", point.EnclosingFunc)
			}
		}
	}
	if !foundTop || !foundFoo {
		t.Fatalf("line match failed: foundTop=%v foundFoo=%v", foundTop, foundFoo)
	}
}
