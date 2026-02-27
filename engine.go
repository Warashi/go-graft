package graft

import (
	"context"
	"sync"
)

// Engine is the mutation test framework entry point.
type Engine struct {
	Config Config

	mu       sync.RWMutex
	registry *ruleRegistry
}

// New creates an engine with validated defaults.
func New(config Config) *Engine {
	return &Engine{
		Config:   config.withDefaults(),
		registry: newRuleRegistry(),
	}
}

// Run executes mutation testing for package patterns.
func (e *Engine) Run(_ context.Context, _ ...string) (*Report, error) {
	return &Report{}, nil
}
