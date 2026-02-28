package graft

import "testing"

func TestConfigWithDefaultsSetsCallGraphModeAuto(t *testing.T) {
	cfg := Config{}
	got := cfg.withDefaults()
	if got.TestSelectionCallGraph != TestSelectionCallGraphAuto {
		t.Fatalf("TestSelectionCallGraph = %q, want %q", got.TestSelectionCallGraph, TestSelectionCallGraphAuto)
	}
}

func TestConfigWithDefaultsNormalizesInvalidCallGraphModeToAuto(t *testing.T) {
	cfg := Config{
		TestSelectionCallGraph: TestSelectionCallGraphMode("unknown"),
	}
	got := cfg.withDefaults()
	if got.TestSelectionCallGraph != TestSelectionCallGraphAuto {
		t.Fatalf("TestSelectionCallGraph = %q, want %q", got.TestSelectionCallGraph, TestSelectionCallGraphAuto)
	}
}

func TestConfigWithDefaultsKeepsExplicitCallGraphMode(t *testing.T) {
	cases := []struct {
		name string
		mode TestSelectionCallGraphMode
	}{
		{name: "auto", mode: TestSelectionCallGraphAuto},
		{name: "rta", mode: TestSelectionCallGraphRTA},
		{name: "cha", mode: TestSelectionCallGraphCHA},
		{name: "ast", mode: TestSelectionCallGraphAST},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				TestSelectionCallGraph: tc.mode,
			}
			got := cfg.withDefaults()
			if got.TestSelectionCallGraph != tc.mode {
				t.Fatalf("TestSelectionCallGraph = %q, want %q", got.TestSelectionCallGraph, tc.mode)
			}
		})
	}
}
