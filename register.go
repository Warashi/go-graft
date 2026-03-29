package graft

import (
	"go/ast"

	"github.com/Warashi/go-graft/internal/rule"
)

// RuleOption configures rule registration.
type RuleOption = rule.Option

// WithName sets report-visible rule name.
func WithName(name string) RuleOption {
	return rule.WithName(name)
}

// WithDeepCopy marks the rule as requiring deep copy input.
func WithDeepCopy() RuleOption {
	return rule.WithDeepCopy()
}

// Register installs a generic mutation rule.
func Register[T ast.Node](e *Engine, mutate func(c *Context, n T) (T, bool), opts ...RuleOption) {
	if e == nil {
		panic("graft: engine must not be nil")
	}
	rule.Register(e.registry, mutate, opts...)
}
