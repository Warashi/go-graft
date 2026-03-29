package testselect

import (
	"fmt"
	"strings"

	"github.com/Warashi/go-graft/internal/model"
	"github.com/Warashi/go-graft/internal/project"
)

// CallGraphMode controls call graph backend selection in test selector internals.
type CallGraphMode string

const (
	CallGraphModeAuto CallGraphMode = "auto"
	CallGraphModeRTA  CallGraphMode = "rta"
	CallGraphModeCHA  CallGraphMode = "cha"
	CallGraphModeAST  CallGraphMode = "ast"
)

type SelectorOptions struct {
	CallGraphMode CallGraphMode
}

type selectorBackend interface {
	name() string
	candidateTests(point model.MutationPoint) []model.TestRef
}

func NewSelectorWithOptions(project *project.Project, tests []model.TestRef, opts SelectorOptions) *Selector {
	mode := opts.CallGraphMode.withDefault()
	out := &Selector{
		project: project,
		tests:   append([]model.TestRef(nil), tests...),
	}
	for _, backendKind := range backendBuildOrder(mode) {
		backend, err := buildBackend(backendKind, project, tests)
		if err != nil {
			out.buildFailures = append(out.buildFailures, fmt.Sprintf("%s: %v", backendKind, err))
			continue
		}
		out.backend = backend
		out.backendName = backend.name()
		break
	}
	if out.backend == nil {
		out.backend = newASTBackend(project, tests)
		out.backendName = out.backend.name()
	}
	return out
}

func (s *Selector) ResolvedBackend() string {
	if s == nil {
		return ""
	}
	return s.backendName
}

func (s *Selector) BuildFailures() []string {
	if s == nil {
		return nil
	}
	return append([]string(nil), s.buildFailures...)
}

func (m CallGraphMode) withDefault() CallGraphMode {
	switch strings.ToLower(strings.TrimSpace(string(m))) {
	case string(CallGraphModeRTA):
		return CallGraphModeRTA
	case string(CallGraphModeCHA):
		return CallGraphModeCHA
	case string(CallGraphModeAST):
		return CallGraphModeAST
	case string(CallGraphModeAuto):
		return CallGraphModeAuto
	default:
		return CallGraphModeAuto
	}
}

func backendBuildOrder(mode CallGraphMode) []string {
	switch mode.withDefault() {
	case CallGraphModeRTA:
		return []string{"rta", "cha", "ast"}
	case CallGraphModeCHA:
		return []string{"cha", "ast"}
	case CallGraphModeAST:
		return []string{"ast"}
	default:
		return []string{"rta", "cha", "ast"}
	}
}

func buildBackend(kind string, project *project.Project, tests []model.TestRef) (selectorBackend, error) {
	switch kind {
	case "rta":
		return newRTABackend(project, tests)
	case "cha":
		return newCHABackend(project, tests)
	case "ast":
		return newASTBackend(project, tests), nil
	default:
		return nil, fmt.Errorf("unknown backend kind %q", kind)
	}
}

type astBackend struct {
	tests          []model.TestRef
	astCallersBack map[functionKey][]functionKey
}

func newASTBackend(project *project.Project, tests []model.TestRef) *astBackend {
	return &astBackend{
		tests:          append([]model.TestRef(nil), tests...),
		astCallersBack: buildReverseCallers(project),
	}
}

func (b *astBackend) name() string {
	return "ast"
}

func (b *astBackend) candidateTests(point model.MutationPoint) []model.TestRef {
	if b == nil {
		return nil
	}
	return candidateTestsByReachability(b.tests, point, b.astCallersBack)
}
