package graft

import (
	"go/ast"
	"reflect"
)

type ruleMutateFunc func(c *Context, n ast.Node) (ast.Node, bool)

type registeredRule struct {
	name       string
	targetType reflect.Type
	deepCopy   bool
	mutate     ruleMutateFunc
}

type ruleRegistry struct {
	ordered []registeredRule
	byType  map[reflect.Type][]registeredRule
	nextID  int
}

func newRuleRegistry() *ruleRegistry {
	return &ruleRegistry{
		byType: make(map[reflect.Type][]registeredRule),
	}
}

func (r *ruleRegistry) add(rule registeredRule) {
	r.ordered = append(r.ordered, rule)
	r.byType[rule.targetType] = append(r.byType[rule.targetType], rule)
}

func (r *ruleRegistry) rulesFor(node ast.Node) []registeredRule {
	if node == nil {
		return nil
	}
	return append([]registeredRule(nil), r.byType[reflect.TypeOf(node)]...)
}

// RuleOption configures rule registration.
type RuleOption func(*ruleConfig)

type ruleConfig struct {
	name     string
	deepCopy bool
}

// WithName sets report-visible rule name.
func WithName(name string) RuleOption {
	return func(cfg *ruleConfig) {
		cfg.name = name
	}
}

// WithDeepCopy marks the rule as requiring deep copy input.
func WithDeepCopy() RuleOption {
	return func(cfg *ruleConfig) {
		cfg.deepCopy = true
	}
}

// Register installs a generic mutation rule.
func Register[T ast.Node](e *Engine, mutate func(c *Context, n T) (T, bool), opts ...RuleOption) {
	if e == nil {
		panic("graft: engine must not be nil")
	}
	if mutate == nil {
		panic("graft: mutate callback must not be nil")
	}

	cfg := ruleConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	targetType := reflect.TypeFor[T]()
	if cfg.name == "" {
		cfg.name = "rule#" + itoa(e.nextRuleID())
	}

	wrapped := registeredRule{
		name:       cfg.name,
		targetType: targetType,
		deepCopy:   cfg.deepCopy,
		mutate: func(c *Context, n ast.Node) (ast.Node, bool) {
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
	}

	e.mu.Lock()
	e.registry.add(wrapped)
	e.mu.Unlock()
}

func (e *Engine) nextRuleID() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.registry.nextID++
	return e.registry.nextID
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
