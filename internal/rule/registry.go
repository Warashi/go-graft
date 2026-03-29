package rule

import (
	"go/ast"
	"reflect"
	"slices"
	"sync"
)

type MutateFunc func(c *Context, n ast.Node) (ast.Node, bool)

type Definition struct {
	Name       string
	TargetType reflect.Type
	DeepCopy   bool
	Mutate     MutateFunc
}

type Snapshot struct {
	ordered []Definition
	byType  map[reflect.Type][]Definition
}

type Registry struct {
	mu      sync.RWMutex
	ordered []Definition
	byType  map[reflect.Type][]Definition
	nextID  int
}

func NewRegistry() *Registry {
	return &Registry{
		byType: make(map[reflect.Type][]Definition),
	}
}

func (r *Registry) Add(def Definition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ordered = append(r.ordered, def)
	r.byType[def.TargetType] = append(r.byType[def.TargetType], def)
}

func (r *Registry) NextID() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	return r.nextID
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := Snapshot{
		ordered: append([]Definition(nil), r.ordered...),
		byType:  make(map[reflect.Type][]Definition, len(r.byType)),
	}
	for key, rules := range r.byType {
		out.byType[key] = append([]Definition(nil), rules...)
	}
	return out
}

func (s Snapshot) RulesFor(node ast.Node) []Definition {
	if node == nil {
		return nil
	}
	return append([]Definition(nil), s.byType[reflect.TypeOf(node)]...)
}

func (s Snapshot) TargetTypes() []reflect.Type {
	types := make([]reflect.Type, 0, len(s.byType))
	for t := range s.byType {
		types = append(types, t)
	}
	slices.SortFunc(types, func(a reflect.Type, b reflect.Type) int {
		switch {
		case a.String() < b.String():
			return -1
		case a.String() > b.String():
			return 1
		default:
			return 0
		}
	})
	return types
}
