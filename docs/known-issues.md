---
title: "Known Issues"
description: "Current implementation gaps, repo-health problems, and behavioral caveats for get-rich-quick."
status: "canonical"
updated: "2026-04-03"
tags: [known-issues, limitations]
---

# Known Issues

This page is intentionally blunt. It exists so contributors and operators do not lose time assuming the happy path is more complete than it really is.

## Repository health

### Unresolved merge conflicts are present

The repository currently contains merge-conflict markers in multiple backend and frontend files, including runtime, risk, API tests, and UI pages/components.

Impact:

- broad `go test ./...` and frontend verification can fail before reaching your area of work
- some docs necessarily describe intended behavior plus current caveats
- the realtime and settings-related UI surfaces are especially likely to need revalidation after conflict resolution

## Product and control-plane gaps

### WebSocket authentication is not enforced

`GET /ws` is mounted outside the auth-protected API group. The current handler allows upgrade and subscription commands without the JWT/API-key middleware used for `/api/v1/*`.

Impact:

- treat the WebSocket feed as an internal/local surface until this is fixed

### Settings edits are in-memory only

The settings page and `PUT /api/v1/settings` update a memory-backed settings service. They do not rewrite `.env`, do not persist to the database, and do not survive restart.

Impact:

- the UI behaves like a control surface, but today it is a runtime-session editor

### There is no user registration flow

Login exists. Registration does not. Local users must be inserted directly into Postgres.

Impact:

- onboarding for anything outside local/dev workflows is incomplete

## Runtime and execution caveats

### Backtest capability exists below the product surface

There is substantial backtest code under `internal/backtest`, and there is a backtest comparison helper under `internal/api/backtest_comparison.go`, but the main API server does not currently expose a supported backtest route set.

Impact:

- backtesting is not yet a first-class documented user workflow

### Polymarket support is incomplete

`polymarket` exists as a market type and there is a Polymarket execution package, but the main production strategy runner does not present live Polymarket execution as a complete, operator-friendly supported path.

Impact:

- treat Polymarket as partial support, not finished support

### Social and news coverage are uneven

The `DataProvider` abstraction includes OHLCV, fundamentals, news, and social sentiment, but not every provider implements every surface. `FINNHUB_*` settings are parsed into config but not currently wired into the runtime provider factory. A `newsapi` package exists but is not part of the main runtime chain.

Impact:

- “feature exists in interface” does not always mean “feature is active in production runtime wiring”

### Whole-pipeline timeout is not currently enforced

The runtime uses phase-specific timeouts, but the pipeline-wide timeout helper currently resolves to no overall timeout.

Impact:

- a run can rely on per-phase limits without a single global wall-clock cap

## UI caveats

### Realtime page is not trustworthy until conflicts are resolved

`web/src/pages/realtime-page.tsx` and several related components currently contain merge conflicts.

Impact:

- do not treat the realtime page as stable product truth until the merge state is cleaned up

### Structured strategy editing needs revalidation

Several strategy config editor components currently have conflict markers.

Impact:

- the underlying API and typed config model are more trustworthy than the current UI editor state

## Documentation caveats

### Older design docs can overstate maturity

`docs/design/` contains valuable architecture intent, but parts of it describe the target system more cleanly than the currently wired system deserves.

Impact:

- prefer [Reference](reference/README.md) for implementation truth
- use design docs for rationale and direction

## Practical advice

Before debugging anything complicated:

1. Check whether the file area is currently in a conflicted state.
2. Verify the route or page is actually mounted in the current server/router.
3. Confirm whether the feature is persisted or merely in-memory.
4. Confirm whether the provider/integration is present only in config/types or actually instantiated in runtime wiring.
