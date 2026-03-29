# go-graft Documentation Ownership Map

This document defines where each kind of information should live.
To keep docs MECE, write full details in the source-of-truth (SSOT) file and keep other files as summaries with links.

## Ownership Table

| Topic | Source of truth (SSOT) | Referencing docs | Writing rule |
| --- | --- | --- | --- |
| Public API, config defaults, report model | [`docs/public-api.md`](public-api.md) | `README.md`, `AGENTS.md`, architecture docs | Keep exported API contracts in one place; other docs summarize and link. |
| Architecture, feature map, execution order | [`docs/architecture.md`](architecture.md) | `README.md`, `AGENTS.md`, `docs/design-summary.md` | Keep cross-feature structure and flow in one place; summaries should link back here. |
| Rule registration and callback semantics | [`docs/rule.md`](rule.md) | `README.md`, `docs/design.md` | Keep rule-specific behavior and constraints in one place. |
| Test discovery, call resolution, test selection behavior | [`docs/selection.md`](selection.md) | `README.md`, `docs/design-pruning-summary.md`, `docs/design.md` | Keep selection rules and backend fallback behavior here. |
| Mutation-point collection, AST clone behavior, overlay building | [`docs/mutation.md`](mutation.md) | `README.md`, `docs/design.md` | Keep mutation artifact generation behavior here. |
| Runtime execution, statuses, and summary semantics | [`docs/execution.md`](execution.md) | `README.md`, `AGENTS.md`, `docs/design.md` | Keep execution and reliability semantics here. |
| User-facing introduction, project maturity, quick start entry point | [`README.md`](../README.md) | `CONTRIBUTING.md`, `SECURITY.md` | Keep onboarding-level explanation in `README.md`; other docs reference it instead of restating. |
| Contribution workflow and pre-PR checks | [`CONTRIBUTING.md`](../CONTRIBUTING.md) | `README.md`, PR template | Keep contribution process in one place; other docs only link. |
| Vulnerability reporting process | [`SECURITY.md`](../SECURITY.md) | `README.md`, `CONTRIBUTING.md` | Keep security contact/reporting steps only in `SECURITY.md`. |
| Repository-invariant agent constraints | [`AGENTS.md`](../AGENTS.md) | agent tooling only | Keep only always-true, repo-specific constraints in `AGENTS.md`; implementation details belong to the feature docs above. |

## Maintenance Rules

1. When adding a new cross-cutting topic, first decide a single SSOT file and then add/update this map.
2. If content in a referencing doc grows beyond summary level, move details back to the SSOT and replace with links.
3. Use section links when possible.
