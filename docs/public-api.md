# go-graft Public API

This document is the source of truth for exported library behavior in package `graft`.

## Engine and Config

```go
type Config struct {
    Workers                int
    MutantTimeout          time.Duration
    BaseTempDir            string
    KeepTemp               bool
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

Defaults:

- `Workers <= 0` -> `1`
- `MutantTimeout <= 0` -> `30s`
- invalid or empty `TestSelectionCallGraph` -> `auto`

## Rule registration

```go
func Register[T ast.Node](e *Engine, mutate func(c *Context, n T) (T, bool), opts ...RuleOption)
func WithName(name string) RuleOption
func WithDeepCopy() RuleOption
```

Behavior:

- One rule targets one concrete `ast.Node` type `T`.
- A mutant is generated only when the callback returns `(mutatedNode, true)`.
- Default callback input is a shallow copy of the matched node.
- `WithDeepCopy()` deep-copies the callback subtree and enables descendant lookup through `Context.Original` and `Context.TypeOf`.
- If deep-copy preparation fails, that mutant is emitted as immediate `Errored`.

## Swap helpers

```go
func RegisterFunctionCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption)
func RegisterMethodCallSwap[F any](e *Engine, from F, to F, opts ...RuleOption)
```

Constraints:

- Function swap accepts package-level functions only.
- Function `from` and `to` must be in the same package and have identical signatures.
- Method swap accepts method expressions, not method values.
- Method `from` and `to` must share the same receiver named type and signature.

## Rule callback context

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

## Report model

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
