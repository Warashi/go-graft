# go-graft Architecture

This document is the architecture index for the current implementation.
Behavioral details live in the feature-specific source-of-truth docs linked below.

## Feature map

The implementation is organized by feature:

1. `project`: load the main-module package graph and source metadata.
2. `rule`: own mutation rule registration, callback context, and swap helper rules.
3. `selection`: discover tests, resolve calls, and choose tests for each mutation point.
4. `mutation`: collect mutation points and build per-mutant overlay artifacts.
5. `execution`: run mutants and summarize execution results.

Shared infrastructure is intentionally minimal:

- `astclone`: AST clone and replacement primitives shared by `rule` and `mutation`

## End-to-end flow

`Engine.Run` executes the pipeline in this order:

1. Load project graph with `project.Loader`.
2. Discover mutation-test candidates and build selector state with `selection`.
3. Collect mutation points with `mutation`.
4. Apply registered rules and build mutant overlays with `mutation` plus `astclone`.
5. Execute selected tests and summarize outcomes with `execution`.

## Source-of-truth index

- Public API and defaults: `docs/public-api.md`
- Rule registration and callback semantics: `docs/rule.md`
- Test discovery and test selection: `docs/selection.md`
- Mutation-point collection and overlay building: `docs/mutation.md`
- Runtime execution and status semantics: `docs/execution.md`
