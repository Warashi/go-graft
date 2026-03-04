# go-graft Architecture Summary

## Purpose

Source of truth: [docs/design.md §1 Scope](design.md#1-scope).

This document is a short reading guide for the current architecture.
Exact implementation behavior is defined in `docs/design.md`.

## Recommended Reading Order

Source of truth: [docs/design.md §2 Public API](design.md#2-public-api), [§3 Core Pipeline](design.md#3-core-pipeline), [§4 Module Details](design.md#4-module-details), [§5 Reliability and Status Semantics](design.md#5-reliability-and-status-semantics), and [§6 Current Limitations](design.md#6-current-limitations).

1. Read [§2 Public API](design.md#2-public-api) for library contracts.
2. Read [§3 Core Pipeline](design.md#3-core-pipeline) to understand overall execution flow.
3. Read [§4 Module Details](design.md#4-module-details) for component responsibilities.
4. Read [§5 Reliability and Status Semantics](design.md#5-reliability-and-status-semantics) and [§6 Current Limitations](design.md#6-current-limitations) for interpretation boundaries.

## High-Level Flow Map

Source of truth: [docs/design.md §3 Core Pipeline](design.md#3-core-pipeline).

The execution flow is:

1. `internal/projectload`
2. `internal/testdiscover`
3. `internal/mutationpoint`
4. `internal/mutantbuild`
5. `internal/testselect`
6. `internal/runner`
7. `internal/reporting`

For responsibilities and exact behavior of each stage, use [§4 Module Details](design.md#4-module-details).

## Quick Index by Topic

Source of truth: [docs/design.md](design.md).

| Topic | Canonical section |
| --- | --- |
| Public API and defaults | [§2 Public API](design.md#2-public-api) |
| Mutant generation invariant and replacement model | [§4.4 Rule application and AST replacement](design.md#44-rule-application-and-ast-replacement-in-enginerun) |
| Test-selection behavior | [§4.6 `internal/testselect`](design.md#46-internaltestselect) |
| Runner command semantics | [§4.7 `internal/runner`](design.md#47-internalrunner) |
| Status meaning and mutation score semantics | [§5 Reliability and Status Semantics](design.md#5-reliability-and-status-semantics) |
| Known limitations | [§6 Current Limitations](design.md#6-current-limitations) |
