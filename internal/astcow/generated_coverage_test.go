package astcow

import (
	"fmt"
	"go/importer"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type generatedFieldKind int

const (
	generatedFieldNone generatedFieldKind = iota
	generatedFieldNode
	generatedFieldSlice
	generatedFieldSliceNode
	generatedFieldMap
	generatedFieldMapNode
)

type generatedFieldMeta struct {
	Name string
	Kind generatedFieldKind
}

type generatedTypeMeta struct {
	Name   string
	Fields []generatedFieldMeta
}

func TestGeneratedCopyReplaceIsUpToDate(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), "copy_replace.go")
	cmd := exec.Command("go", "run", "./cmd/astcowgen", "-out", tempFile)
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run ./cmd/astcowgen failed: %v\n%s", err, string(output))
	}

	want, err := os.ReadFile("copy_replace.go")
	if err != nil {
		t.Fatalf("ReadFile(copy_replace.go) error = %v", err)
	}
	got, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", tempFile, err)
	}

	if string(got) != string(want) {
		t.Fatal("generated copy_replace.go is stale; run `go generate ./internal/astcow`")
	}
}

func TestGeneratedCopyReplaceCoversGoAstNodes(t *testing.T) {
	t.Parallel()

	srcBytes, err := os.ReadFile("copy_replace.go")
	if err != nil {
		t.Fatalf("ReadFile(copy_replace.go) error = %v", err)
	}
	src := string(srcBytes)

	meta, err := collectGeneratedMetaFromGoAst()
	if err != nil {
		t.Fatalf("collectGeneratedMetaFromGoAst() error = %v", err)
	}

	shallowFn := extractFunction(src, "func shallowCopy(")
	replaceFn := extractFunction(src, "func replaceChild(")
	if shallowFn == "" {
		t.Fatal("missing shallowCopy function in generated file")
	}
	if replaceFn == "" {
		t.Fatal("missing replaceChild function in generated file")
	}

	for _, typ := range meta {
		shallowCase := extractCaseBody(t, shallowFn, typ.Name)
		replaceCase := extractCaseBody(t, replaceFn, typ.Name)

		for _, field := range typ.Fields {
			switch field.Kind {
			case generatedFieldSlice, generatedFieldSliceNode:
				mustContain(t, shallowCase, "cloned."+field.Name+" = append(")
			case generatedFieldMap, generatedFieldMapNode:
				mustContain(t, shallowCase, "cloned."+field.Name+" = cloneMap(")
			}

			switch field.Kind {
			case generatedFieldNode:
				mustContain(t, replaceCase, "replaceField(&p."+field.Name+", oldChild, newChild)")
			case generatedFieldSliceNode:
				mustContain(t, replaceCase, "replaceSlice(p."+field.Name+", oldChild, newChild)")
			case generatedFieldMapNode:
				mustContain(t, replaceCase, "replaceMapValue(p."+field.Name+", oldChild, newChild)")
			}
		}
	}
}

func collectGeneratedMetaFromGoAst() ([]generatedTypeMeta, error) {
	pkg, err := importer.Default().Import("go/ast")
	if err != nil {
		return nil, err
	}

	nodeObj := pkg.Scope().Lookup("Node")
	if nodeObj == nil {
		return nil, fmt.Errorf("go/ast.Node not found")
	}
	nodeIface, ok := nodeObj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("go/ast.Node is not interface")
	}

	names := pkg.Scope().Names()
	sort.Strings(names)

	out := make([]generatedTypeMeta, 0, len(names))
	for _, name := range names {
		obj := pkg.Scope().Lookup(name)
		tn, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := tn.Type().(*types.Named)
		if !ok {
			continue
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		if !types.AssignableTo(types.NewPointer(named), nodeIface) {
			continue
		}

		fields := make([]generatedFieldMeta, 0, st.NumFields())
		for field := range st.Fields() {
			fields = append(fields, generatedFieldMeta{
				Name: field.Name(),
				Kind: classifyGeneratedField(field.Type(), nodeIface),
			})
		}
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})

		out = append(out, generatedTypeMeta{Name: name, Fields: fields})
	}

	return out, nil
}

func classifyGeneratedField(t types.Type, nodeIface *types.Interface) generatedFieldKind {
	if types.AssignableTo(t, nodeIface) {
		return generatedFieldNode
	}
	if sl, ok := t.Underlying().(*types.Slice); ok {
		if types.AssignableTo(sl.Elem(), nodeIface) {
			return generatedFieldSliceNode
		}
		return generatedFieldSlice
	}
	if mp, ok := t.Underlying().(*types.Map); ok {
		if types.AssignableTo(mp.Elem(), nodeIface) {
			return generatedFieldMapNode
		}
		return generatedFieldMap
	}
	return generatedFieldNone
}

func extractFunction(src string, prefix string) string {
	start := strings.Index(src, prefix)
	if start < 0 {
		return ""
	}
	end := strings.Index(src[start+len(prefix):], "\nfunc ")
	if end < 0 {
		return src[start:]
	}
	return src[start : start+len(prefix)+end]
}

func extractCaseBody(t *testing.T, fnSrc string, typeName string) string {
	t.Helper()

	caseHeader := "\tcase *ast." + typeName + ":"
	_, after, ok := strings.Cut(fnSrc, caseHeader)
	if !ok {
		t.Fatalf("missing case for %s", typeName)
	}
	rest := after

	nextCase := strings.Index(rest, "\n\tcase *ast.")
	defaultCase := strings.Index(rest, "\n\tdefault:")
	end := len(rest)
	if nextCase >= 0 && nextCase < end {
		end = nextCase
	}
	if defaultCase >= 0 && defaultCase < end {
		end = defaultCase
	}
	return rest[:end]
}

func mustContain(t *testing.T, body string, want string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("case body missing %q", want)
	}
}
