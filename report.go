package graft

// Status is the execution result of one mutant.
type Status int

const (
	Killed Status = iota
	Survived
	Unsupported
	Errored
)

type ExecutedPackage struct {
	ImportPath string
	RunPattern string
}

type MutantResult struct {
	ID          string
	RuleName    string
	File        string
	Line        int
	Column      int
	Package     string
	Executed    []ExecutedPackage
	Status      Status
	Reason      string
	Command     []string
	Stdout      string
	Stderr      string
	TimedOut    bool
	ElapsedNsec int64
}

// Report is the final mutation test summary.
type Report struct {
	Total       int
	Killed      int
	Survived    int
	Unsupported int
	Errored     int
	Mutants     []MutantResult
}

// MutationScore returns killed/(killed+survived).
func (r Report) MutationScore() float64 {
	denom := r.Killed + r.Survived
	if denom == 0 {
		return 0
	}
	return float64(r.Killed) / float64(denom)
}
