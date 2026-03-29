package graft

import "github.com/Warashi/go-graft/internal/rule"

// Context is passed to each rule callback.
type Context = rule.Context

func newContext() *Context {
	return rule.NewContext()
}
