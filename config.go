package graft

import "time"

// Config controls execution behavior.
type Config struct {
	Workers       int
	MutantTimeout time.Duration
	BaseTempDir   string
	KeepTemp      bool
}

func (c Config) withDefaults() Config {
	out := c
	if out.Workers <= 0 {
		out.Workers = 1
	}
	if out.MutantTimeout <= 0 {
		out.MutantTimeout = 30 * time.Second
	}
	return out
}
