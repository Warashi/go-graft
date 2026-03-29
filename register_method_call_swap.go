package graft

import "github.com/Warashi/go-graft/internal/rule"

// RegisterMethodCallSwap installs a method-call swap rule using method expressions.
func RegisterMethodCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption) {
	if e == nil {
		panic("graft: engine must not be nil")
	}
	rule.RegisterMethodCallSwap(e.registry, from, to, opts...)
}
