# go-graft Detailed Design (Current Implementation)

## 1. Scope

This document describes the current implementation of `github.com/Warashi/go-graft`.
It is not a roadmap. Future ideas are intentionally omitted unless already implemented.

### In scope

- Library behavior and public API in package `graft`
- Internal execution pipeline and data flow
- Current test-selection and runner behavior
- Current limits and reliability boundaries

### Out of scope

- CLI design (the project currently provides no CLI)
- Unimplemented optimization proposals
- Compatibility promises beyond the current `v0` phase

## 2. Public API

### 2.1 Engine and Config

```go
type Config struct {
    Workers       int
    MutantTimeout time.Duration
    BaseTempDir   string
    KeepTemp      bool
    TestSelectionCallGraph TestSelectionCallGraphMode
}

type TestSelectionCallGraphMode string

const (
    TestSelectionCallGraphAuto TestSelectionCallGraphMode = "auto"
    TestSelectionCallGraphRTA  TestSelectionCallGraphMode = "rta"
    TestSelectionCallGraphCHA  TestSelectionCallGraphMode = "cha"
    TestSelectionCallGraphAST  TestSelectionCallGraphMode = "ast"
)

type Engine struct {
    Config Config
}

func New(config Config) *Engine
func (e *Engine) Run(runCtx context.Context, patterns ...string) (*Report, error)
```

Defaulting behavior:

- `Workers <= 0` -> `1`
- `MutantTimeout <= 0` -> `30s`
- `TestSelectionCallGraph` invalid/empty -> `auto`

### 2.2 Rule registration

```go
type RuleOption func(*ruleConfig)

func Register[T ast.Node](e *Engine, mutate func(c *Context, n T) (T, bool), opts ...RuleOption)
func WithName(name string) RuleOption
func WithDeepCopy() RuleOption
```

Current behavior:

- One rule targets one concrete `ast.Node` type `T`.
- A mutant is created only when callback returns `(mutatedNode, true)`.
- `WithDeepCopy()` is currently stored as rule metadata, but does not alter execution behavior yet.

### 2.3 Swap helpers

```go
func RegisterFunctionCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption)
func RegisterMethodCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption)
```

Current constraints:

- Function swap accepts package-level functions only.
- Function `from` and `to` must be in the same package and have identical signatures.
- Method swap accepts method expressions (not method values).
- Method `from` and `to` must share the same receiver named type and signature.

### 2.4 Rule callback context

```go
type Context struct {
    Fset  *token.FileSet
    Pkg   *packages.Package
    File  *ast.File
    Types *types.Info
    Path  []ast.Node
}

func (c *Context) Original(node ast.Node) ast.Node
func (c *Context) TypeOf(node ast.Node) types.Type
```

`Original` resolves cloned nodes back to original AST nodes through internal clone tracking.
`TypeOf` consults `types.Info` via that original-node mapping.

### 2.5 Report model

```go
type Status int

const (
    Killed Status = iota
    Survived
    Unsupported
    Errored
)

type Report struct {
    Total       int
    Killed      int
    Survived    int
    Unsupported int
    Errored     int
    Mutants     []MutantResult
}

func (r Report) MutationScore() float64
```

`MutationScore()` is `killed / (killed + survived)`.
`Unsupported` and `Errored` are excluded from the denominator.

## 3. Core Pipeline

The implementation flow is:

1. `internal/projectload`
2. `internal/testdiscover`
3. `internal/mutationpoint`
4. `internal/mutantbuild`
5. `internal/testselect`
6. `internal/runner`
7. `internal/reporting`

`Engine.Run` orchestrates this flow.

## 4. Module Details

### 4.1 `internal/projectload`

Responsibilities:

- Load package graph with `go/packages` and `Tests: true`.
- Request syntax, types, type info, imports, dependencies, module metadata.
- Keep only packages in the main module.
- Normalize source paths and build lookup maps.

Important detail:

- Both `GoFiles` and `CompiledGoFiles` are tracked.
- Overlay replacement keys are based on compiled file paths when available.

### 4.2 `internal/testdiscover`

Responsibilities:

- Discover top-level test functions:
  - Name prefix `Test` with uppercase continuation
  - Single parameter of type `*testing.T` (validated with AST + type info)
- Build intra-project function call relations
- Detect whether each test can reach `(*graft.Engine).Run`

Filtering behavior:

- Default: tests that can reach engine `Run` are excluded as mutation tests.
- Directive overrides:
  - `//gograft:include` forces inclusion.
  - `//gograft:exclude` forces exclusion.
- Included and excluded tests are returned with explicit exclusion reasons.

### 4.3 `internal/mutationpoint`

Responsibilities:

- Traverse non-test Go files (`*_test.go` excluded).
- Collect nodes whose concrete type matches any registered rule target type.
- Capture mutation metadata:
  - package ID/import path
  - source file path
  - AST path from file root to target node
  - source position
  - enclosing function (if any)
  - compiled file path mapping

### 4.4 Rule application and AST replacement in `Engine.Run`

Per mutation point and matching rule:

1. Shallow-copy target node with `astcow.ShallowCopyNode`.
2. Build callback context (`Fset`, package, file, types info, path).
3. Execute rule callback through `applyRule`, which recovers panics and converts them to errors.
4. If callback changed node:
   - clone path and replace one node with `astcow.ClonePath`
   - build file-level mutant AST
5. Emit immediate `Errored` result when any step fails.

Invariant:

- One generated mutant corresponds to one mutation point and one node replacement.

### 4.5 `internal/mutantbuild`

Responsibilities:

- Create per-mutant temp directory (`graft-<id>-*` prefix when ID is available).
- Create:
  - `overlay/` for mutated files
  - `tmp/` for mutant-local `TMPDIR`
- Format mutated AST via `go/format` and write file.
- Write `overlay.json`:

```json
{
  "Replace": {
    "<original-compiled-go-file>": "<mutated-file-path>"
  }
}
```

### 4.6 `internal/testselect`

Responsibilities:

1. Build one selector at `Engine.Run` start and reuse it for all mutation points.
2. Resolve one backend chain from `Config.TestSelectionCallGraph`:
   - `auto`: `rta -> cha -> ast`
   - `rta`: `rta -> cha -> ast`
   - `cha`: `cha -> ast`
   - `ast`: `ast`
3. Use mutation-point enclosing function as reachability seed.
4. Collect candidate tests reachable through reverse calls.
5. Fallback to all discovered tests when no candidate is found.
6. Prune candidates by reverse package dependencies from mutant package.
7. Group selected tests as `map[importPath][]testName` with sorted unique names.

Design choice:

- Fallback behavior intentionally avoids over-pruning when static call analysis misses edges.

### 4.7 `internal/runner`

Responsibilities:

- Execute mutants using worker pool (`Workers`).
- For each mutant:
  - If selected test groups are empty, return `Unsupported`.
  - Execute selected packages in sorted order.
  - Build package-specific `-run` regex using escaped, sorted test names.
  - Run `go test` with fixed options:

```text
go test <pkg> -run <regex> -failfast -parallel=1 -count=1 -overlay=<overlay.json>
```

Runtime behavior:

- `TMPDIR` is set to mutant temp `tmp/`.
- Timeout is enforced per mutant package command via `context.WithTimeout`.
- First failing package marks mutant as `Killed` and stops that mutant run.
- Timeout is represented as `Killed` with `TimedOut=true`.
- Temp directory is removed unless `KeepTemp=true`.

### 4.8 `internal/reporting` and report composition

Responsibilities:

- Summarize counts by internal mutant status.
- Map internal statuses to public `Status`.
- Emit per-mutant details:
  - location (`file`, `line`, `column`, package)
  - rule name and ID
  - executed packages and run patterns
  - failure command/output
  - timeout and elapsed time

## 5. Reliability and Status Semantics

Status intent:

- `Killed`: mutation was detected by selected tests (including timeout-induced failure).
- `Survived`: selected tests all passed.
- `Unsupported`: no reliable verdict (for example, empty selection).
- `Errored`: framework failed to build or prepare mutant execution.

Important rule:

- `Unsupported` is never merged into `Survived`.
- This separation avoids false confidence in mutation score.

## 6. Current Limitations

- Call analysis prefers RTA/CHA but may still fall back to AST when deeper analysis cannot be safely built.
- Dynamic behaviors (reflection, advanced indirection patterns, complex dispatch) can reduce precision.
- `WithDeepCopy()` is not yet behaviorally applied in the execution pipeline.
- Mutation is limited to one-node replacement per generated mutant.

## 7. Debug and Development Notes

- Set `GO_GRAFT_DEBUG` to print mutation-test auto-exclusion details.
- `GO_GRAFT_DEBUG` also prints the selected test-selection call graph backend and fallback reasons.
- Set `Config{KeepTemp: true}` to keep mutant temp directories for reproduction/debugging.
- Standard repository checks:
  - `go vet ./...`
  - `go test ./...`
