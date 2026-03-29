# AGENTS.md for go-graft

This file records repository-specific facts for `github.com/Warashi/go-graft`.
Do not place user-level global preferences or rules for other projects here.

## Project Overview

- The project is currently `v0` (pre-`v1.0.0`), so breaking changes are allowed while APIs and behavior are stabilized.
- `go-graft` is a library-only Go mutation testing framework built on top of `go test -overlay`.
- Documentation ownership is defined in `docs/documentation-map.md`.

## Source-of-Truth References

- Public API, defaults, and behavior contracts: `docs/public-api.md`.
- Architecture and feature boundaries: `docs/architecture.md`.
- Rule registration and callback semantics: `docs/rule.md`.
- Test discovery and selection semantics: `docs/selection.md`.
- Mutation collection and overlay building: `docs/mutation.md`.
- Status semantics and runtime execution boundaries: `docs/execution.md`.

## Standard Development Checks

- lint: `go vet ./...`
- test: `go test ./...`

## AGENTS.md Maintenance Requirements

Update this `AGENTS.md` and `docs/documentation-map.md` in the same change whenever any of the following is modified:

- Responsibilities of major directories or the core processing flow
- Default library behavior (major config defaults or execution flow)
- Execution status categories or runner execution method
- Standard lint/test check commands
- Documentation ownership boundaries between files

When updating, keep only facts that are always true for this repository, and omit temporary operational notes.
