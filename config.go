package graft

import (
	"strings"
	"time"
)

// TestSelectionCallGraphMode controls the call graph backend for test selection.
type TestSelectionCallGraphMode string

const (
	TestSelectionCallGraphAuto TestSelectionCallGraphMode = "auto"
	TestSelectionCallGraphRTA  TestSelectionCallGraphMode = "rta"
	TestSelectionCallGraphCHA  TestSelectionCallGraphMode = "cha"
	TestSelectionCallGraphAST  TestSelectionCallGraphMode = "ast"
)

func (m TestSelectionCallGraphMode) withDefault() TestSelectionCallGraphMode {
	switch strings.ToLower(strings.TrimSpace(string(m))) {
	case string(TestSelectionCallGraphRTA):
		return TestSelectionCallGraphRTA
	case string(TestSelectionCallGraphCHA):
		return TestSelectionCallGraphCHA
	case string(TestSelectionCallGraphAST):
		return TestSelectionCallGraphAST
	case string(TestSelectionCallGraphAuto):
		return TestSelectionCallGraphAuto
	default:
		return TestSelectionCallGraphAuto
	}
}

// Config controls execution behavior.
type Config struct {
	Workers                int
	MutantTimeout          time.Duration
	BaseTempDir            string
	KeepTemp               bool
	TestSelectionCallGraph TestSelectionCallGraphMode
}

func (c Config) withDefaults() Config {
	out := c
	if out.Workers <= 0 {
		out.Workers = 1
	}
	if out.MutantTimeout <= 0 {
		out.MutantTimeout = 30 * time.Second
	}
	out.TestSelectionCallGraph = out.TestSelectionCallGraph.withDefault()
	return out
}
