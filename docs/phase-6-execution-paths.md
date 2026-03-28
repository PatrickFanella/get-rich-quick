---
title: Phase 6 Execution Paths
type: tracking
created: 2026-03-27
tags: [tracking, phase-6, frontend, cli, testing, hardening]
---

# Phase 6: Execution Paths

> 17 issues across 6 tracks. **10 ready now**, 7 blocked.
> Updated: 2026-03-27
> Tracking issue: [#358](https://github.com/PatrickFanella/get-rich-quick/issues/358)

## Context

Phase 5 delivers the API layer, CI/CD, and paper trading validation. Phase 6 builds the React dashboard, CLI/TUI, comprehensive test suites, and performs security hardening. This is the final phase before the system is production-ready.

**Prerequisites from Phase 5:** #74 (REST API), #77 (WebSocket), #88 (JWT auth) must be complete before frontend work begins.

## Summary

| Track | Name                 | Total  | Ready  | Blocked | Models          |
| ----- | -------------------- | :----: | :----: | :-----: | --------------- |
| A     | React Frontend       |   7    |   1    |    6    | Mixed           |
| B     | CLI & TUI            |   2    |   2    |    0    | GPT-5.4         |
| C     | Testing              |   3    |   2    |    1    | Mixed           |
| D     | Security & Hardening |   1    |   0    |    1    | Claude Opus 4.6 |
| E     | Documentation        |   2    |   2    |    0    | GPT-5.4         |
| F     | Market Adapters      |   2    |   1    |    1    | Claude Opus 4.6 |
|       | **Total**            | **17** | **10** |  **7**  |                 |

---

## Track A: React Frontend

> Depends on: Phase 5 #74 (REST API), #77 (WebSocket), #88 (JWT auth)

| #   | Issue                                                               | Title                                     | Size | Blocker     | Status  | Model           |
| --- | ------------------------------------------------------------------- | ----------------------------------------- | :--: | ----------- | ------- | --------------- |
| 1   | [#97](https://github.com/PatrickFanella/get-rich-quick/issues/97)   | Scaffold React frontend with Vite         |  M   | Phase 5 API | READY   | GPT-5.4         |
| 2   | [#100](https://github.com/PatrickFanella/get-rich-quick/issues/100) | Implement Dashboard page                  |  L   | #97         | BLOCKED | Claude Opus 4.6 |
| 3   | [#92](https://github.com/PatrickFanella/get-rich-quick/issues/92)   | Implement Strategy Management page        |  L   | #97         | BLOCKED | Claude Opus 4.6 |
| 4   | [#95](https://github.com/PatrickFanella/get-rich-quick/issues/95)   | Implement Portfolio page                  |  L   | #97         | BLOCKED | Claude Opus 4.6 |
| 5   | [#98](https://github.com/PatrickFanella/get-rich-quick/issues/98)   | Implement Pipeline Run visualization page |  L   | #97, #77    | BLOCKED | Claude Opus 4.6 |
| 6   | [#99](https://github.com/PatrickFanella/get-rich-quick/issues/99)   | Implement Memories page with search       |  M   | #97         | BLOCKED | GPT-5.4         |
| 7   | [#101](https://github.com/PatrickFanella/get-rich-quick/issues/101) | Implement Settings page                   |  M   | #97         | BLOCKED | GPT-5.4         |

**Tech stack (from design docs):**

- React 19, Vite 6, React Router 7
- TanStack Query 5 (server state), React Hook Form 7 + Zod 3
- shadcn/ui components, Tailwind CSS 4
- Recharts 2 (portfolio/P&L), Lightweight Charts 4 (candlestick/price)
- TanStack Table 8

**Page → API mapping:**

| Page                | Key Endpoints                                                       | WebSocket Events                                             |
| ------------------- | ------------------------------------------------------------------- | ------------------------------------------------------------ |
| Dashboard           | `GET /portfolio/summary`, `GET /strategies`                         | `signal`, `position_update`                                  |
| Strategy Management | `GET/POST /strategies`, `PUT /strategies/:id`                       | —                                                            |
| Portfolio           | `GET /portfolio/positions`, `GET /trades`, `GET /portfolio/history` | `order_filled`, `position_update`                            |
| Pipeline Run        | `GET /runs/:id`, `GET /runs/:id/decisions`                          | `pipeline_start`, `agent_decision`, `debate_round`, `signal` |
| Memories            | `GET /memories/search`, `DELETE /memories/:id`                      | —                                                            |
| Settings            | `GET/PUT /config`, `GET/PUT /risk/limits`                           | —                                                            |

**Execution order:** #97 scaffold first (routing, auth, layout, query client), then #100, #92, #95 in parallel (independent pages), then #98 (needs WebSocket), #99, #101.

---

## Track B: CLI & TUI

> Depends on: Nothing (fully independent)

| #   | Issue                                                               | Title                                   | Size | Blocker | Status | Model   |
| --- | ------------------------------------------------------------------- | --------------------------------------- | :--: | ------- | ------ | ------- |
| 1   | [#102](https://github.com/PatrickFanella/get-rich-quick/issues/102) | Implement CLI with Cobra commands       |  M   | None    | READY  | GPT-5.4 |
| 2   | [#103](https://github.com/PatrickFanella/get-rich-quick/issues/103) | Implement TUI dashboard with Bubble Tea |  L   | #102    | READY  | GPT-5.4 |

**CLI commands (expected):**

- `tradingagent serve` — start API + scheduler
- `tradingagent run <strategy-id>` — trigger single pipeline run
- `tradingagent strategies list|create|delete`
- `tradingagent backtest run|compare`
- `tradingagent risk status|kill|reset`

**Execution order:** #102 first (Cobra command skeleton + serve command), then #103 (Bubble Tea TUI wrapping CLI functionality).

---

## Track C: Testing

> Depends on: Phase 5 #74 (API) for smoke test

| #   | Issue                                                               | Title                                             | Size | Blocker     | Status  | Model           |
| --- | ------------------------------------------------------------------- | ------------------------------------------------- | :--: | ----------- | ------- | --------------- |
| 1   | [#104](https://github.com/PatrickFanella/get-rich-quick/issues/104) | Write comprehensive unit test suite               |  L   | None        | READY   | GPT-5.4         |
| 2   | [#105](https://github.com/PatrickFanella/get-rich-quick/issues/105) | Write integration test suite with real PostgreSQL |  L   | None        | READY   | Claude Opus 4.6 |
| 3   | [#113](https://github.com/PatrickFanella/get-rich-quick/issues/113) | Implement end-to-end smoke test                   |  M   | Phase 5 API | BLOCKED | Claude Opus 4.6 |

**Current test coverage:** 30+ test files exist across agent, debate, risk, trader, scheduler, backtest, config, and LLM packages. These issues cover gaps and add structured suites.

**#104 scope:** Identify untested functions via `go test -cover`, add tests for data providers, execution brokers, repository layer, memory, and any gaps in existing packages.

**#105 scope:** Tests that hit a real PostgreSQL instance (via Docker Compose or CI service container). Cover: repository CRUD operations, migration correctness, concurrent pipeline run persistence, market data cache behavior.

**#113 scope:** Full end-to-end: start API → create strategy → trigger run → verify pipeline completes → check persisted decisions and signal. Uses `httptest` or real server.

**Execution order:** #104 and #105 in parallel (independent), then #113 after API exists.

---

## Track D: Security & Hardening

> Depends on: All major code written (should be one of the last items)

| #   | Issue                                                               | Title                        | Size | Blocker        | Status  | Model           |
| --- | ------------------------------------------------------------------- | ---------------------------- | :--: | -------------- | ------- | --------------- |
| 1   | [#108](https://github.com/PatrickFanella/get-rich-quick/issues/108) | Security audit and hardening |  L   | Tracks A, B, C | BLOCKED | Claude Opus 4.6 |

**Scope (from issue + deployment docs):**

- Input validation audit (API handlers, WebSocket messages)
- SQL injection review (parameterized queries via pgx)
- Secret handling (env vars only, no hardcoded keys)
- JWT token expiry and refresh flow
- CORS configuration (restrict to frontend origin)
- Rate limiting (100 req/min per client)
- Database user permissions (no superuser)
- Dependency audit (`go mod tidy`, check for known CVEs)
- Kill switch file path traversal check

**Execution order:** After all feature code is merged. This is a review pass, not a feature.

---

## Track E: Documentation

> Depends on: Nothing (can start anytime, finalized after code stabilizes)

| #   | Issue                                                               | Title                                                 | Size | Blocker | Status | Model   |
| --- | ------------------------------------------------------------------- | ----------------------------------------------------- | :--: | ------- | ------ | ------- |
| 1   | [#109](https://github.com/PatrickFanella/get-rich-quick/issues/109) | Create comprehensive README and developer setup guide |  M   | None    | READY  | GPT-5.4 |
| 2   | [#106](https://github.com/PatrickFanella/get-rich-quick/issues/106) | Create operational runbooks                           |  M   | None    | READY  | GPT-5.4 |

**#109 scope:** Quick-start guide, prerequisites, `docker-compose up`, environment variables, running tests, project structure overview. Replace current minimal README.

**#106 scope:** Operational runbooks for: deploying updates, triggering/stopping the kill switch, investigating failed pipeline runs, resetting circuit breaker, database backup/restore, handling stuck cron jobs.

**Execution order:** #109 first (needed for any new contributor), #106 after deployment topology is finalized.

---

## Track F: Market Adapters

> Depends on: Phase 5 #89 (SL/TP enforcement) for full integration

| #   | Issue                                                               | Title                                                  | Size | Blocker | Status | Model           |
| --- | ------------------------------------------------------------------- | ------------------------------------------------------ | :--: | ------- | ------ | --------------- |
| 1   | [#107](https://github.com/PatrickFanella/get-rich-quick/issues/107) | Implement Polymarket broker adapter                    |  L   | None    | READY  | Claude Opus 4.6 |
| 2   | [#347](https://github.com/PatrickFanella/get-rich-quick/issues/347) | Migrate production nodes to typed phase I/O interfaces |  L   | None    | READY  | Claude Opus 4.6 |

**#107 notes:** High-risk. Polymarket uses a CLOB (central limit order book) on Polygon L2. Requires: wallet integration, EIP-712 signed orders, conditional token framework. ADR-006 sets 50 bps fixed slippage and 5% max per-market exposure (ADR-008 position limits). Paper-only initially.

**#347 notes:** Carried from Phase 5. Migrate `BaseAnalyst`, `BullResearcher`, `BearResearcher`, `Trader`, `RiskManager` to typed interfaces (`AnalystNode`, `DebaterNode`, `TraderNode`, `RiskJudgeNode`). Infrastructure exists; requires refactoring `PromptBuilder` signatures.

---

## Dependency Graph

```
Phase 5 prerequisites (#74 API, #77 WebSocket, #88 JWT)
  │
  ├── Track A #97 (Scaffold React) ─── ready once Phase 5 API exists
  │     ├── #100 (Dashboard) ──────────┐
  │     ├── #92 (Strategy Mgmt) ───────┤
  │     ├── #95 (Portfolio) ───────────┤── all blocked by #97
  │     ├── #98 (Pipeline Viz) ────────┤   #98 also needs #77 (WebSocket)
  │     ├── #99 (Memories) ────────────┤
  │     └── #101 (Settings) ───────────┘
  │
  └── Track C #113 (Smoke test) ─── blocked by API

Track B #102 (CLI) ─── independent
  └── #103 (TUI) ─── blocked by #102

Track C #104 (Unit tests) ─── independent
Track C #105 (Integration tests) ─── independent

Track E #109 (README) ─── independent
Track E #106 (Runbooks) ─── independent

Track F #107 (Polymarket) ─── independent
Track F #347 (Typed I/O) ─── independent

Track D #108 (Security audit) ─── blocked by all feature tracks
```

---

## Recommended Wave Execution

### Wave 1 — Independent work, no blockers (8 issues)

- Track A: **#97** (scaffold React frontend — can start once Phase 5 API is stable)
- Track B: **#102** (CLI with Cobra)
- Track C: **#104** (unit test suite), **#105** (integration tests with PostgreSQL)
- Track E: **#109** (README), **#106** (operational runbooks)
- Track F: **#107** (Polymarket broker adapter)
- Track F: **#347** (typed phase I/O migration)

### Wave 2 — Frontend pages + TUI (7 issues)

- Track A: **#100** (Dashboard), **#92** (Strategy Management), **#95** (Portfolio) — parallel, all blocked by #97
- Track A: **#99** (Memories), **#101** (Settings) — parallel
- Track B: **#103** (TUI with Bubble Tea) — blocked by #102
- Track C: **#113** (smoke test) — blocked by Phase 5 API

### Wave 3 — Pipeline visualization + security (2 issues)

- Track A: **#98** (Pipeline Run visualization) — needs WebSocket (#77), most complex frontend page
- Track D: **#108** (Security audit) — after all major code is merged

---

## Key Design Principles

1. **Frontend is a consumer, not a blocker** — The React dashboard depends on the API layer from Phase 5 but blocks nothing else. It can be built in parallel with testing and documentation.
2. **CLI before TUI** — Cobra CLI provides the command structure that Bubble Tea wraps. CLI is also useful for scripting and CI even without the TUI.
3. **Security audit last** — Auditing before all code is written means re-auditing. Do it once, after all feature tracks merge.
4. **Polymarket is high-risk, low-priority** — The adapter requires blockchain wallet integration (EIP-712 signatures, Polygon L2). Paper-only initially. Don't let it block other work.
5. **Test suites fill gaps** — 30+ test files already exist. #104 and #105 are about coverage completeness, not starting from scratch.

## Model Recommendations

| Complexity | Model           | Use For                                                                                                                                        |
| ---------- | --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| High       | Claude Opus 4.6 | Dashboard (#100), Strategy Mgmt (#92), Portfolio (#95), Pipeline Viz (#98), Integration tests (#105), Security audit (#108), Polymarket (#107) |
| Medium     | GPT-5.4         | Scaffold (#97), Memories (#99), Settings (#101), CLI (#102), TUI (#103), Unit tests (#104), README (#109), Runbooks (#106)                     |
