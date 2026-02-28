package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Warashi/go-graft/internal/model"
)

type Runner struct {
	Workers       int
	MutantTimeout time.Duration
	KeepTemp      bool
}

func (r Runner) Run(ctx context.Context, mutants []model.Mutant) []model.MutantExecResult {
	if len(mutants) == 0 {
		return nil
	}
	workers := r.Workers
	if workers <= 0 {
		workers = 1
	}
	timeout := r.MutantTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	type job struct {
		index  int
		mutant model.Mutant
	}
	jobs := make(chan job)
	results := make(chan struct {
		index  int
		result model.MutantExecResult
	})

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Go(func() {
			for j := range jobs {
				res := runOne(ctx, j.mutant, timeout, r.KeepTemp)
				results <- struct {
					index  int
					result model.MutantExecResult
				}{index: j.index, result: res}
			}
		})
	}

	go func() {
		for idx, mutant := range mutants {
			jobs <- job{index: idx, mutant: mutant}
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	out := make([]model.MutantExecResult, len(mutants))
	for item := range results {
		out[item.index] = item.result
	}
	return out
}

func runOne(parent context.Context, mutant model.Mutant, timeout time.Duration, keepTemp bool) model.MutantExecResult {
	started := time.Now()
	result := model.MutantExecResult{
		Mutant:       mutant,
		Status:       model.MutantSurvived,
		ExecutedPkgs: make(map[string]string),
	}
	defer func() {
		result.ElapsedNsec = time.Since(started).Nanoseconds()
		if keepTemp || mutant.TempDir == "" {
			return
		}
		if err := os.RemoveAll(mutant.TempDir); err != nil {
			result.InternalErrMsg = err.Error()
		}
	}()

	if len(mutant.SelectedTests.ByImportPath) == 0 {
		result.Status = model.MutantUnsupported
		result.Reason = "test selection produced 0 tests"
		return result
	}

	pkgs := sortedKeys(mutant.SelectedTests.ByImportPath)
	for _, pkg := range pkgs {
		tests := mutant.SelectedTests.ByImportPath[pkg]
		if len(tests) == 0 {
			continue
		}
		runPattern := buildRunPattern(tests)
		result.ExecutedPkgs[pkg] = runPattern

		args := buildGoTestArgs(pkg, runPattern, mutant.OverlayPath)
		cmdCtx, cancel := context.WithTimeout(parent, timeout)
		cmd := exec.CommandContext(cmdCtx, "go", args...)
		if mutant.WorkDir != "" {
			cmd.Dir = mutant.WorkDir
		}
		cmd.Env = append(os.Environ(), "TMPDIR="+mutant.OverlayTmpDir)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		cancel()

		if err == nil {
			continue
		}

		result.Status = model.MutantKilled
		result.Reason = "go test returned non-zero"
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.FailedCommand = append([]string{"go"}, args...)
		if cmdCtx.Err() == context.DeadlineExceeded {
			result.Reason = "mutant execution timed out"
			result.TimedOut = true
		}
		return result
	}

	return result
}

func buildGoTestArgs(pkg string, runPattern string, overlayPath string) []string {
	return []string{
		"test",
		pkg,
		"-run", runPattern,
		"-failfast",
		"-parallel=1",
		"-count=1",
		"-overlay=" + overlayPath,
	}
}

func buildRunPattern(testNames []string) string {
	names := append([]string(nil), testNames...)
	slices.Sort(names)
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		quoted = append(quoted, regexp.QuoteMeta(name))
	}
	return "^(" + strings.Join(quoted, "|") + ")$"
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func DebugCommandString(args []string) string {
	return fmt.Sprintf("go %s", strings.Join(args, " "))
}
