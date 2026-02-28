# go-graft

`go-graft` is a Go mutation testing framework built on top of `go test -overlay`.
It can run mutants in parallel without rewriting your original source files.

## Project status

- Current maturity: `v0` (pre-1.0)
- Breaking changes may happen while APIs and behavior are being stabilized.

## Quick start (library)

```go
package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"time"

	"github.com/Warashi/go-graft"
)

func main() {
	e := graft.New(graft.Config{
		Workers:       2,
		MutantTimeout: 30 * time.Second,
	})

	graft.Register[*ast.BinaryExpr](e, func(_ *graft.Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
		if n.Op != token.ADD {
			return nil, false
		}
		n.Op = token.SUB
		return n, true
	}, graft.WithName("add-to-sub"))

	report, err := e.Run(context.Background(), "./...")
	if err != nil {
		panic(err)
	}
	fmt.Printf("score=%.2f\n", report.MutationScore())
}
```

## Execution statuses

Each mutant is reported as one of the following statuses:

- `Killed`: at least one selected test failed
- `Survived`: selected tests passed, mutation was not detected
- `Unsupported`: analysis/execution could not produce a reliable verdict
- `Errored`: mutant generation or execution failed unexpectedly

## Public API surface

Representative public API entry points (non-exhaustive):

- `Engine`
- `Register`
- `RegisterMethodCallSwap`
- `RegisterFunctionCallSwap`
- `Context`
- `Report`

For the canonical, complete API definition, see:

- `docs/design.md` (Section 2: Public API)
- `go doc -all github.com/Warashi/go-graft`

## Compatibility policy

This repository is currently in `v0`.
Backward compatibility is **not guaranteed** before `v1.0.0`.

## FAQ

### Why does go-graft use `go test -overlay`?

It allows running mutants without modifying your original source files.

### Why are some mutants marked `Unsupported`?

`Unsupported` is used when the tool cannot provide a reliable result.
This is intentionally separated from `Survived` to avoid false confidence.

### How do I debug mutant execution?

Set `Config{KeepTemp: true}` and inspect temporary mutant directories and command output.

### How is the test-selection call graph chosen?

Use `Config{TestSelectionCallGraph: ...}`:

- `auto` (default): `rta -> cha -> ast`
- `rta`: `rta -> cha -> ast`
- `cha`: `cha -> ast`
- `ast`: AST-only analysis

When a backend cannot be used safely, go-graft falls back to the next backend and keeps the "no candidate => all discovered tests" safety rule.

## Design docs

- [Current architecture summary](docs/design-summary.md)
- [Current test selection strategy](docs/design-pruning-summary.md)
- [Current detailed design](docs/design.md)

## Contributing and security

- [Contributing guide](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## License

Apache License 2.0. See [LICENSE](LICENSE).
