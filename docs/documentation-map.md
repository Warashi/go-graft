# go-graft Documentation Ownership Map

This document defines where each kind of information should live.
To keep docs MECE, write full details in the source-of-truth (SSOT) file and keep other files as summaries with links.

## Ownership Table

| Topic | Source of truth (SSOT) | Referencing docs | Writing rule |
| --- | --- | --- | --- |
| Implementation behavior, API contracts, statuses, pipeline, constraints | [`docs/design.md`](design.md) | `README.md`, `docs/design-summary.md`, `docs/design-pruning-summary.md`, `AGENTS.md` | Keep exact behavior definitions only in `docs/design.md`; other docs should summarize and link to section anchors. |
| User-facing introduction, project maturity, quick start entry point | [`README.md`](../README.md) | `CONTRIBUTING.md`, `SECURITY.md` | Keep onboarding-level explanation in `README.md`; other docs reference it instead of restating. |
| Contribution workflow and pre-PR checks | [`CONTRIBUTING.md`](../CONTRIBUTING.md) | `README.md`, PR template | Keep contribution process in one place; other docs only link. |
| Vulnerability reporting process | [`SECURITY.md`](../SECURITY.md) | `README.md`, `CONTRIBUTING.md` | Keep security contact/reporting steps only in `SECURITY.md`. |
| Repository-invariant agent constraints | [`AGENTS.md`](../AGENTS.md) | agent tooling only | Keep only always-true, repo-specific constraints in `AGENTS.md`; implementation details belong to `docs/design.md`. |

## Maintenance Rules

1. When adding a new cross-cutting topic, first decide a single SSOT file and then add/update this map.
2. If content in a referencing doc grows beyond summary level, move details back to the SSOT and replace with links.
3. Use section links when possible (for example, `docs/design.md#2-public-api`).
