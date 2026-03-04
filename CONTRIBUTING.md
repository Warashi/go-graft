# Contributing to go-graft

Thanks for your interest in contributing.

## Scope

This document covers contribution workflow only.
For product behavior and architecture references, use:

- `README.md`
- `docs/design.md`
- `docs/documentation-map.md`

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

Please use the bug report issue form.
The required fields are defined in `.github/ISSUE_TEMPLATE/bug_report.yml`.

## Code review expectations

- Prefer minimal, focused changes.
- Explain tradeoffs for non-trivial design decisions.
- Keep CI green.

## Security issues

Do not open public issues for security vulnerabilities.
Use the process in `SECURITY.md` (source of truth).
