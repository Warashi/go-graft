# go-graft Selection Feature

This document is the source of truth for test discovery and test selection behavior.

## Test discovery

`selection.DiscoverDetailed`:

- finds top-level `TestXxx(*testing.T)` functions
- validates `*testing.T` using AST plus type information
- excludes tests that can reach `(*graft.Engine).Run`

Directive overrides:

- `//gograft:include` forces inclusion
- `//gograft:exclude` forces exclusion
- include wins when both appear

## Call resolution

Selection resolves top-level function calls by:

1. trying `types.Info` first
2. falling back to syntax-level import alias resolution
3. preferring the current package ID when one import path maps to multiple package IDs

## Selector backends

Supported modes:

- `auto`: `rta -> cha -> ast`
- `rta`: `rta -> cha -> ast`
- `cha`: `cha -> ast`
- `ast`: `ast`

If backend construction fails, fallback reasons are recorded and the next backend is tried.

## Per-mutation behavior

For one mutation point, selection:

1. uses the enclosing function as the reachability seed when available
2. finds candidate tests via the resolved backend
3. falls back to all discovered tests when reachability returns none
4. prunes by reverse package dependencies from the mutant package
5. groups selected tests as `map[importPath][]testName` with sorted unique names
