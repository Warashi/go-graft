# Contributing to go-graft

Thanks for your interest in contributing.

## Scope

- This project is currently in `v0`.
- Breaking changes are allowed while APIs and behavior are being stabilized.
- Public API candidates are documented in `AGENTS.md`.

## Development setup

1. Install Go `1.26.x`.
2. Clone the repository.
3. Run checks:

```bash
go vet ./...
go test ./...
```

## Pull request workflow

1. Create a branch from `main`.
2. Keep commits small and use [Conventional Commits](https://www.conventionalcommits.org/).
3. Add or update tests when behavior changes.
4. Run `go vet ./...` and `go test ./...` before opening a PR.
5. Update docs when public behavior or library behavior changes.

## Reporting bugs

Please use the bug report issue form and include:

- Reproduction steps
- Expected behavior
- Actual behavior
- `go version`
- Command(s) you executed

## Code review expectations

- Prefer minimal, focused changes.
- Explain tradeoffs for non-trivial design decisions.
- Keep CI green.

## Security issues

Do not open public issues for security vulnerabilities.
Use the process in `SECURITY.md`.
