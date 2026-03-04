# go-graft

`go-graft` is a Go mutation testing framework built on top of `go test -overlay`.
It can run mutants in parallel without rewriting your original source files.

## Project status

- Current maturity: `v0` (pre-1.0)
- Breaking changes may happen while APIs and behavior are being stabilized.
- Most of this repository was initially generated with coding agents.
- Treat the project as experimental and validate behavior, security, and performance for your use case.

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

Each mutant is classified into one status.
For exact definitions and scoring semantics, see:

- [Detailed design: Report model](docs/design.md#25-report-model)
- [Detailed design: Reliability and status semantics](docs/design.md#5-reliability-and-status-semantics)

## Public API surface

Use these references as the canonical API definitions:

- [Detailed design: Public API](docs/design.md#2-public-api)
- `go doc -all github.com/Warashi/go-graft`

## Compatibility policy

This repository is currently in `v0`.
Backward compatibility is **not guaranteed** before `v1.0.0`.

## FAQ

### Why does go-graft use `go test -overlay`?

It allows running mutants without modifying your original source files.

### Why are some mutants marked `Unsupported`?

See [Detailed design: Reliability and status semantics](docs/design.md#5-reliability-and-status-semantics).

### How do I debug mutant execution?

Set `Config{KeepTemp: true}` and inspect temporary mutant directories and command output.

### How is the test-selection call graph chosen?

See [Detailed design: `internal/testselect`](docs/design.md#46-internaltestselect) for backend selection and fallback behavior.

## Design docs

- [Documentation ownership map](docs/documentation-map.md)
- [Current architecture summary](docs/design-summary.md)
- [Current test selection strategy](docs/design-pruning-summary.md)
- [Current detailed design](docs/design.md)

## Contributing and security

- [Contributing guide](CONTRIBUTING.md)
- [Security policy](SECURITY.md)

## License

Apache License 2.0. See [LICENSE](LICENSE).
