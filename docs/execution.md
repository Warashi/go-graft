# go-graft Execution Feature

This document is the source of truth for mutant execution and status semantics.

## Runner behavior

The execution feature:

- runs mutants with a worker pool
- executes selected packages in sorted order
- uses one `go test` command per selected package
- stops a mutant run at the first failing package

The command shape is:

```text
go test <pkg> -run <regex> -failfast -parallel=1 -count=1 -overlay=<overlay.json>
```

Runtime details:

- `TMPDIR` is set to the mutant-local temp directory
- timeout is enforced per mutant package command
- temp directories are removed unless `KeepTemp=true`

## Status semantics

- `Killed`: selected tests detected the mutant, including timeout-induced failure
- `Survived`: all selected tests passed
- `Unsupported`: no reliable verdict, currently used when test selection produced zero tests
- `Errored`: framework-side preparation or build failure prevented execution

`Unsupported` is never merged into `Survived`.

## Report summary

Execution summary counts are derived from internal mutant statuses and then mapped into the public `Report`.
