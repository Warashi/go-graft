# Test Selection Strategy (Current Implementation)

## Objective

The selection strategy balances:

- avoiding false negatives (missing a test that could kill a mutant), and
- reducing unnecessary test execution.

The current implementation intentionally favors reliability over aggressive pruning.

## Stage 1: Base Test Discovery Filter (`internal/testdiscover`)

Before per-mutant selection begins, go-graft discovers top-level tests and filters mutation tests:

- Includes normal top-level `TestXxx(*testing.T)` functions.
- Excludes tests that can reach `(*graft.Engine).Run` through static call traversal.
- Allows explicit overrides:
  - `//gograft:include` forces inclusion.
  - `//gograft:exclude` forces exclusion.

This prevents dogfooding mutation tests from recursively invoking the engine unless explicitly allowed.

## Stage 2: Per-Mutant Reachability (`internal/testselect`)

For each mutation point:

1. Build one selector per engine run with a configured backend chain:
   - `auto` (default): `rta -> cha -> ast`
   - `rta`: `rta -> cha -> ast`
   - `cha`: `cha -> ast`
   - `ast`: AST only
2. Use the enclosing function of the mutation point as the reachability seed.
3. Walk reverse callers to collect reachable tests.

If no tests are found through this graph, selection falls back to all discovered tests.
This fallback avoids over-pruning from analysis blind spots.

## Stage 3: Reverse Dependency Pruning (`internal/testselect`)

After candidate tests are collected:

1. Build reverse package dependencies from loaded package imports.
2. Keep only tests in:
   - the mutant package, or
   - packages that depend (directly or transitively) on the mutant package.

Tests in unrelated packages are safely pruned.

## Stage 4: Execution Grouping (`internal/testselect` + `internal/runner`)

- Selected tests are grouped by package import path.
- Test names are sorted, deduplicated, and converted to a `-run` regex.
- Runner executes package groups sequentially inside one mutant, while mutants run in parallel by worker pool.

If no package/test entries remain, runner reports the mutant as `Unsupported` with reason `test selection produced 0 tests`.

## Tradeoffs and Limits

- RTA/CHA setup can fail on some package shapes; selector then falls back to the next backend and eventually AST.
- AST call resolution remains intentionally simple (`Ident` and imported `SelectorExpr`) and is used as final fallback.
- Dynamic dispatch patterns can still reduce precision even with CHA/RTA.
- To avoid risky false confidence, the implementation prefers fallback expansion or `Unsupported` over claiming `Survived` on weak evidence.
