# Test Selection Strategy (Intent and Tradeoffs)

## Objective

Source of truth: [docs/design.md §4.6 `internal/testselect`](design.md#46-internaltestselect).

This summary explains why test selection exists and which tradeoffs guide its design.
Exact backend order, fallback behavior, and filtering rules are intentionally defined only in `docs/design.md`.

## Strategy Intent

Source of truth: [docs/design.md §4.6 `internal/testselect`](design.md#46-internaltestselect) and [§4.2 `internal/testdiscover`](design.md#42-internaltestdiscover).

The strategy is designed to:

- reduce unnecessary test execution when safe,
- avoid missing tests that could kill mutants, and
- prevent recursive/self-referential mutation-test execution in normal cases.

## Reliability-First Policy

Source of truth: [docs/design.md §4.6 `internal/testselect`](design.md#46-internaltestselect) and [§5 Reliability and Status Semantics](design.md#5-reliability-and-status-semantics).

When static analysis confidence is insufficient, the implementation chooses safer expansion behavior rather than aggressive pruning.
This policy prioritizes verdict reliability over raw speed.

## Main Tradeoffs

Source of truth: [docs/design.md §4.6 `internal/testselect`](design.md#46-internaltestselect), [§6 Current Limitations](design.md#6-current-limitations), and [§4.7 `internal/runner`](design.md#47-internalrunner).

1. Precision vs. coverage:
   tighter pruning can reduce runtime but raises false-negative risk if reachability misses edges.
2. Analysis cost vs. portability:
   richer call-graph analysis can improve precision but may not be available for all package shapes.
3. Pruning gains vs. operational simplicity:
   grouping and selective execution reduce work, but conservative fallback remains necessary for safety.

## Operator Guidance

Source of truth: [docs/design.md §2.1 Engine and Config](design.md#21-engine-and-config), [§4.6 `internal/testselect`](design.md#46-internaltestselect), and [§7 Debug and Development Notes](design.md#7-debug-and-development-notes).

Treat this document as rationale only.
For debugging or behavior interpretation, always follow `docs/design.md`.
