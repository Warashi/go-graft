# AGENTS.md for go-graft

This file records repository-specific facts for `github.com/Warashi/go-graft`.
Do not place user-level global preferences or rules for other projects here.

## Project Overview

- The project is currently `v0` (pre-`v1.0.0`), so breaking changes are allowed while APIs and behavior are stabilized.
- `go-graft` is a Go mutation testing framework built on `go test -overlay`.
- The project is provided as a library only (no CLI).
- Public API surface is in the root package: `Engine`, `Register`, `RegisterMethodCallSwap`, `RegisterFunctionCallSwap`, `Context`, and `Report`.

## Implementation Invariants (Current)

- Mutants are built with the assumption `1 mutant = 1 mutation point` (single-node replacement).
- Execution statuses are handled separately as `Killed`, `Survived`, `Unsupported`, and `Errored`.
- Test execution is handled by `internal/runner`, which runs `go test` with `-overlay`, `-failfast`, `-parallel=1`, and `-count=1`.
- `internal/testdiscover` extracts regular tests and auto-excludes mutation tests that can reach `(*graft.Engine).Run`. This can be overridden with `//gograft:include` and `//gograft:exclude` directives.
- The core flow is split by responsibility as:
  `internal/projectload` -> `internal/testdiscover` -> `internal/mutationpoint` -> `internal/mutantbuild` -> `internal/testselect` -> `internal/runner` -> `internal/reporting`.

## Standard Development Checks

- lint: `go vet ./...`
- test: `go test ./...`

## AGENTS.md Maintenance Requirements

Update this `AGENTS.md` in the same change whenever any of the following is modified:

- Responsibilities of major directories or the core processing flow
- Default library behavior (major config defaults or execution flow)
- Execution status categories or runner execution method
- Standard lint/test check commands

When updating, keep only facts that are always true for this repository, and omit temporary operational notes.
