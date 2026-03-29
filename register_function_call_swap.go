package graft

import "github.com/Warashi/go-graft/internal/rule"

// RegisterFunctionCallSwap installs a function-call swap rule using package-level functions.
func RegisterFunctionCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption) {
	if e == nil {
		panic("graft: engine must not be nil")
	}
	rule.RegisterFunctionCallSwap(e.registry, from, to, opts...)
}
