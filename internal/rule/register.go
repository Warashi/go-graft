package rule

import (
	"fmt"
	"go/ast"
	"reflect"
)

type Option func(*config)

type config struct {
	name     string
	deepCopy bool
}

// WithName sets report-visible rule name.
func WithName(name string) Option {
	return func(cfg *config) {
		cfg.name = name
	}
}

// WithDeepCopy marks the rule as requiring deep copy input.
func WithDeepCopy() Option {
	return func(cfg *config) {
		cfg.deepCopy = true
	}
}

// Register installs a generic mutation rule.
func Register[T ast.Node](registry *Registry, mutate func(c *Context, n T) (T, bool), opts ...Option) {
	if registry == nil {
		panic("graft: registry must not be nil")
	}
	if mutate == nil {
		panic("graft: mutate callback must not be nil")
	}

	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	targetType := reflect.TypeFor[T]()
	if cfg.name == "" {
		cfg.name = "rule#" + itoa(registry.NextID())
	}

	registry.Add(Definition{
		Name:       cfg.name,
		TargetType: targetType,
		DeepCopy:   cfg.deepCopy,
		Mutate: func(c *Context, n ast.Node) (ast.Node, bool) {
			typed, ok := n.(T)
			if !ok {
				return nil, false
			}
			mutated, changed := mutate(c, typed)
			if !changed {
				return nil, false
			}
			return mutated, true
		},
	})
}

func Apply(def Definition, ctx *Context, node ast.Node) (mutated ast.Node, changed bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("rule %q panicked: %v", def.Name, recovered)
		}
	}()
	mutated, changed = def.Mutate(ctx, node)
	if changed && mutated == nil {
		return nil, false, fmt.Errorf("rule %q returned nil mutant node", def.Name)
	}
	return mutated, changed, nil
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
