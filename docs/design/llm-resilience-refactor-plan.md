---
title: 'LLM Resilience + Report Worker Refactor Plan'
description: '6-PR implementation plan for provider chain, budget guard, report worker, and observability.'
status: 'approved'
created: '2026-04-16'
tags: [llm, resilience, refactor, plan]
---

# Refactor Plan: LLM Resilience + Report Worker Pipeline

## Current State

**LLM provider chain:** `newLLMProviderFromConfig` → `ThrottledProvider(4)` → `CachedProvider` (in-memory, opt-out via `LLM_CACHE_ENABLED=false`). No retry. No fallback. No multi-provider ladder.

**Pipeline failures:** When the single configured provider times out or 429s, the entire scheduled strategy run fails. There's no recovery path.

**Existing primitives (built, tested, not wired):**

| Primitive                       | Location                                      | Status                                     |
| ------------------------------- | --------------------------------------------- | ------------------------------------------ |
| `RetryProvider`                 | `internal/llm/retry.go`                       | tested, **not wired**                      |
| `FallbackProvider`              | `internal/llm/fallback.go`                    | tested, **not wired**                      |
| `ThrottledProvider`             | `internal/llm/throttled.go`                   | tested, **wired** (concurrency=4)          |
| `CacheProvider`                 | `internal/llm/cache.go`                       | tested, **wired** (in-memory)              |
| `Registry`                      | `internal/llm/registry.go`                    | tested, **not used in runtime**            |
| `JobOrchestrator`               | `internal/automation/orchestrator.go`         | wired, full cron/dedup/deps/persist/status |
| `papervalidation`               | `internal/papervalidation/`                   | library-only, **no runtime job**           |
| `debateTimeoutFallbackProvider` | `cmd/tradingagent/debate_timeout_provider.go` | wired for debate phases only               |

**Report generation:** `papervalidation.GenerateReport` exists as a pure function. Not called from any runtime path. No scheduled job. No API endpoint to fetch reports.

**Runtime entry point:** `cmd/tradingagent/runtime.go` → `newAPIServer()` builds deps including LLM provider, scheduler, automation orchestrator.

**Key config knobs today:**

- `LLM_TIMEOUT=30s` (default)
- `LLM_DEBATE_TIMEOUT` (env, optional)
- `LLM_CACHE_ENABLED` (default true)
- `SCHEDULER_JOB_TIMEOUT=45m` (default)
- `ENABLE_SCHEDULER=false` (default)

## Target State

1. **Resilient LLM provider chain:** `throttle → retry → fallback → cache` wired globally. Multiple providers configurable. Paid-tier providers have out-of-credits degradation path.
2. **Provider ladder (config-driven, not hardcoded):** Primary from `LLM_DEFAULT_PROVIDER`. Fallback from new `LLM_FALLBACK_PROVIDER` env var. Registry used for multi-provider resolution.
3. **Very generous timeouts:** Per-call timeouts high (5m default) during stabilization phase. Pipeline-level timeout from `SCHEDULER_JOB_TIMEOUT` (default 45m). No premature kills.
4. **Report worker:** `papervalidation` report generation wired as an automation job. Reports persisted to DB. Pipeline reads latest snapshot. Non-blocking on stale.
5. **Quota-aware scheduling:** Internal budget counters. Jitter on job start. Backpressure when queue saturated.
6. **Observability:** Metrics for fallback storms, stale age, report success rate. Auto-disable on consecutive failure threshold.

## User Decisions (from interview)

- **Branch 1 (SLA/workload):** P95 < 20m, 45m hard cap, ≤24 reports/day, cache stale OK 5–60m
- **Branch 2 (providers):** No hardcoded providers. OpenRouter free OK but we don't choose model. GitHub Models. Paid if available. Paid must have out-of-credits path. Never auto-spend.
- **Branch 3 (resilience):** Retry on 429/5xx, maxAttempts=2. Fallback on timeout + transport error + 429 burst. **Timeouts very high** while getting pipeline functioning — don't let timeout stop an otherwise successful run.
- **Branch 4 (worker):** Reports as separate async job. Pipeline reads latest snapshot. Non-blocking. Stale report = warning, not failure.
- **Branch 5 (scheduling):** Internal budget counters. Jitter + spread. 1–4h stale tolerance for trading, 24h for discovery.
- **Branch 6 (observability):** 98% success target, fallback < 20%, auto-disable on storm, rollback if queue growth 3 intervals.

## Affected Files

| File                                                   | Change Type | PR  | Dependencies  |
| ------------------------------------------------------ | ----------- | --- | ------------- |
| `internal/llm/provider_chain.go`                       | create      | 1   | —             |
| `internal/llm/provider_chain_test.go`                  | create      | 1   | —             |
| `internal/llm/budget.go`                               | create      | 1   | —             |
| `internal/llm/budget_test.go`                          | create      | 1   | —             |
| `internal/config/config.go`                            | modify      | 2   | PR 1          |
| `internal/config/validate.go`                          | modify      | 2   | PR 1          |
| `internal/config/validate_test.go`                     | modify      | 2   | PR 1          |
| `.env.example`                                         | modify      | 2   | PR 1          |
| `cmd/tradingagent/runtime.go`                          | modify      | 3   | PR 1, 2       |
| `cmd/tradingagent/runtime_test.go`                     | modify      | 3   | PR 1, 2       |
| `migrations/000017_report_artifacts.up.sql`            | create      | 4   | —             |
| `migrations/000017_report_artifacts.down.sql`          | create      | 4   | —             |
| `internal/repository/postgres/report_artifact.go`      | create      | 4   | migration     |
| `internal/repository/postgres/report_artifact_test.go` | create      | 4   | migration     |
| `internal/automation/jobs_reports.go`                  | create      | 5   | PR 3, 4       |
| `internal/automation/jobs_reports_test.go`             | create      | 5   | PR 3, 4       |
| `internal/automation/orchestrator.go`                  | modify      | 5   | PR 5 jobs     |
| `internal/api/report_handlers.go`                      | create      | 5   | PR 4          |
| `internal/api/server.go`                               | modify      | 5   | PR 5 handlers |
| `internal/metrics/metrics.go`                          | modify      | 6   | PR 1, 5       |
| `docs/reference/llm-resilience.md`                     | create      | 6   | all           |

## Execution Plan

---

### PR 1 — LLM Provider Chain + Budget Guard (`internal/llm`)

_Pure library code. Zero runtime changes. Safe to merge independently._

- [ ] **1.1** Create `internal/llm/provider_chain.go`
  - `NewProviderChain(primary Provider, logger, opts...)` builder
  - Compose: `throttle(N) → retry(maxAttempts, backoff) → fallback(secondary) → cache`
  - All wrappers already exist; this is glue code that returns a single `Provider`
  - Options: `WithFallback(Provider)`, `WithRetry(maxAttempts int)`, `WithThrottle(n int)`, `WithCache(ResponseCache)`, `WithBudget(*Budget)`
  - Chain order: outer throttle → retry → fallback → inner cache
  - Per-call timeout option: `WithCallTimeout(duration)` wraps each Complete call in `context.WithTimeout`

- [ ] **1.2** Create `internal/llm/budget.go`
  - `Budget` struct: `{MaxRequestsPerDay int, MaxTokensPerDay int, mu, counters, resetAt}`
  - `Budget.Allow() bool` — returns false when daily budget exhausted
  - `Budget.Record(promptTokens, completionTokens int)`
  - `Budget.Reset()` — called on UTC midnight or manual
  - `BudgetGuardProvider` wrapping `Provider` — returns `ErrBudgetExhausted` when `Allow()` false
  - `ErrBudgetExhausted` sentinel error (not retryable by RetryProvider)

- [ ] **1.3** Create tests for both
  - Table-driven: chain with all wrappers, chain with subset, budget exhaustion, budget reset
  - Verify budget exhaustion is NOT retried (RetryProvider classifies it as client error)
  - Verify: `go test -short -count=1 ./internal/llm/...`

- [ ] **1.4** Verify: `go build ./cmd/tradingagent` still passes (no runtime imports changed)

---

### PR 2 — Config: Multi-Provider + Budget Env Vars

_Config-only. No behavioral change until runtime wired._

- [ ] **2.1** Add to `internal/config/config.go` `LLMConfig` struct + `Load()`:

  ```
  LLM_FALLBACK_PROVIDER    (string, default "")
  LLM_FALLBACK_MODEL       (string, default "")
  LLM_RETRY_MAX_ATTEMPTS   (int, default 2)
  LLM_CALL_TIMEOUT         (duration, default "5m")
  LLM_BUDGET_REQUESTS_DAY  (int, default 0 = unlimited)
  LLM_BUDGET_TOKENS_DAY    (int, default 0 = unlimited)
  LLM_THROTTLE_CONCURRENCY (int, default 4)
  ```

- [ ] **2.2** Add to `internal/config/validate.go`:
  - If `LLM_FALLBACK_PROVIDER` set, validate it's a known provider name
  - If `LLM_FALLBACK_PROVIDER` set, validate its API key is present (skip for ollama)
  - `LLM_RETRY_MAX_ATTEMPTS` must be ≥ 1
  - `LLM_CALL_TIMEOUT` must be > 0
  - `LLM_THROTTLE_CONCURRENCY` must be ≥ 1

- [ ] **2.3** Update `internal/config/validate_test.go`:
  - Fallback provider without key → error
  - Fallback provider ollama → ok
  - Fallback provider empty → ok (no fallback)
  - Bad retry count → error

- [ ] **2.4** Update `.env.example` with new vars + comments:

  ```env
  # LLM resilience (PR: llm-resilience)
  LLM_FALLBACK_PROVIDER=
  LLM_FALLBACK_MODEL=
  LLM_RETRY_MAX_ATTEMPTS=2
  LLM_CALL_TIMEOUT=5m
  LLM_BUDGET_REQUESTS_DAY=0
  LLM_BUDGET_TOKENS_DAY=0
  LLM_THROTTLE_CONCURRENCY=4
  ```

- [ ] **2.5** Verify: `go test -short -count=1 ./internal/config/...` + `go build ./cmd/tradingagent`

---

### PR 3 — Wire Provider Chain in Runtime

_Behavioral change: production runtime now uses retry+fallback+budget._

- [ ] **3.1** Modify `cmd/tradingagent/runtime.go`:
  - Replace `wrapLLMProvider(throttleLLM(newLLMProviderFromConfig(...)), ...)` with:
    ```go
    buildProviderChain(cfg.LLM, appMetrics, logger)
    ```
  - New function `buildProviderChain`:
    1. Build primary via existing `newLLMProviderFromConfig`
    2. If `LLM_FALLBACK_PROVIDER` set, build secondary via `newLLMProviderForSelection`
    3. Compose chain: `llm.NewProviderChain(primary, logger, opts...)`
    4. Options from config: throttle concurrency, retry attempts, fallback, cache, budget, call timeout
  - The existing `throttleLLM` and `wrapLLMProvider` become unused → remove or mark deprecated

- [ ] **3.2** Ensure timeout behavior:
  - Per-call timeout from `LLM_CALL_TIMEOUT` (default **5m** — very high per user request)
  - Job-level timeout from `SCHEDULER_JOB_TIMEOUT` (default 45m) unchanged
  - `LLM_DEBATE_TIMEOUT` + `debateTimeoutFallbackProvider` still respected for debate phases (layered on top of chain)
  - Net effect: calls get 5m each; job gets 45m total; debates get `LLM_DEBATE_TIMEOUT` with quick-model fallback as before

- [ ] **3.3** Update/add tests in `cmd/tradingagent/`:
  - Verify chain builds with primary-only config
  - Verify chain builds with primary+fallback
  - Verify budget guard rejects when exhausted
  - Verify existing smoke tests still pass

- [ ] **3.4** Verify: `go test -short -count=1 ./...` + `go build ./cmd/tradingagent`

---

### PR 4 — Report Artifact Storage (Migration + Repo)

_DB schema + repository. No runtime job yet._

- [ ] **4.1** Create `migrations/000017_report_artifacts.up.sql`:

  ```sql
  CREATE TABLE report_artifacts (
      id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      strategy_id       UUID NOT NULL REFERENCES strategies(id),
      report_type       TEXT NOT NULL DEFAULT 'paper_validation',
      time_bucket       TIMESTAMPTZ NOT NULL,
      status            TEXT NOT NULL DEFAULT 'pending',
      report_json       JSONB,
      provider          TEXT,
      model             TEXT,
      prompt_tokens     INT DEFAULT 0,
      completion_tokens INT DEFAULT 0,
      latency_ms        INT DEFAULT 0,
      error_message     TEXT,
      created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
      completed_at      TIMESTAMPTZ,
      UNIQUE (strategy_id, report_type, time_bucket)
  );
  CREATE INDEX idx_report_artifacts_strategy_type
      ON report_artifacts (strategy_id, report_type, completed_at DESC);
  ```

  Idempotency key = `(strategy_id, report_type, time_bucket)`.

- [ ] **4.2** Create `migrations/000017_report_artifacts.down.sql`:

  ```sql
  DROP TABLE IF EXISTS report_artifacts;
  ```

- [ ] **4.3** Create `internal/repository/postgres/report_artifact.go`:
  - `ReportArtifactRepo` with `Upsert`, `GetLatest(strategyID, reportType)`, `List(filter, limit, offset)`
  - `ReportArtifact` domain struct matching table

- [ ] **4.4** Create `internal/repository/postgres/report_artifact_test.go`:
  - Upsert idempotency (same key → update)
  - GetLatest returns most recent completed
  - GetLatest returns nil when none

- [ ] **4.5** Bump `pgrepo.RequiredSchemaVersion` to 17

- [ ] **4.6** Verify: migration applies cleanly to dev DB, `go test -short -count=1 ./internal/repository/...`

---

### PR 5 — Report Worker Job + API Endpoint

_Wires papervalidation into automation orchestrator + exposes latest report via API._

- [ ] **5.1** Create `internal/automation/jobs_reports.go`:
  - `registerReportJobs()` called from `RegisterAll()`
  - Job: `"paper_validation_report"`, cron `"0 17 * * 1-5"` (5 PM ET daily, after market close)
  - Schedule type: `ScheduleTypeAfterHours`
  - Job function:
    1. List active paper strategies from `StrategyRepo`
    2. For each strategy: load backtest metrics from latest runs
    3. Call `papervalidation.GenerateReport()`
    4. Persist to `report_artifacts` table via repo
    5. On failure: persist error row, continue to next strategy (non-blocking)
  - Jitter: random 0–120s delay before each strategy report to spread load
  - Budget check: if `Budget.Allow()` false, skip LLM-dependent reports, log warning

- [ ] **5.2** Update `internal/automation/orchestrator.go`:
  - Add `ReportArtifactRepo` to `OrchestratorDeps`
  - Add `registerReportJobs()` call in `RegisterAll()`

- [ ] **5.3** Create `internal/api/report_handlers.go`:
  - `GET /api/v1/strategies/{id}/reports/latest` → returns latest completed `report_artifacts` row as JSON
  - `GET /api/v1/strategies/{id}/reports` → paginated list
  - Response includes `stale_seconds` field = `now - completed_at`

- [ ] **5.4** Wire handlers in `internal/api/server.go` route registration

- [ ] **5.5** Wire `ReportArtifactRepo` in `cmd/tradingagent/runtime.go` → `OrchestratorDeps` + `api.Deps`

- [ ] **5.6** Create tests:
  - `internal/automation/jobs_reports_test.go` — mock repos, verify report generation + persist
  - `internal/api/report_handlers_test.go` — handler response shape, stale calculation

- [ ] **5.7** Verify: `go test -short -count=1 ./...` + `go build ./cmd/tradingagent`

---

### PR 6 — Observability, Metrics, Docs, Kill Criteria

_Metrics wiring + operational docs + auto-disable._

- [ ] **6.1** Modify `internal/metrics/metrics.go`:
  - `RecordLLMRetry(provider string)`
  - `RecordLLMBudgetExhausted()`
  - `RecordReportWorkerSuccess(strategyID string)`
  - `RecordReportWorkerError(strategyID string)`
  - `ObserveReportStaleness(seconds float64)`

- [ ] **6.2** Wire metrics into provider chain (PR 1 hooks) and report worker (PR 5 hooks)

- [ ] **6.3** Add auto-disable logic to report worker:
  - Track `ConsecutiveFailures` (already in `RegisteredJob`)
  - If `consecutiveFailures >= 5`: auto-disable job, emit alert via notification manager
  - Manual re-enable via automation API `PUT /api/v1/automation/jobs/{name}/enable`

- [ ] **6.4** Create `docs/reference/llm-resilience.md`:
  - Provider chain architecture diagram
  - Env var reference table
  - Troubleshooting: "all providers down", "budget exhausted", "report stale"
  - Success criteria for 7-day validation:
    - Report worker success rate > 98%
    - Fallback usage < 20%
    - Pipeline timeout failures ↓ 80%
    - P95 report latency < 20m

- [ ] **6.5** Verify: `go test -short -count=1 ./...` + `go build ./cmd/tradingagent` + `golangci-lint run ./...`

---

## Rollback Plan

Each PR is independently revertable:

| PR   | Revert impact                                                                              |
| ---- | ------------------------------------------------------------------------------------------ |
| PR 6 | Remove metrics + docs. No behavioral change.                                               |
| PR 5 | Remove report job + API. Reports stop generating. No pipeline impact.                      |
| PR 4 | `DROP TABLE report_artifacts`. No runtime references remain after PR 5 reverted.           |
| PR 3 | Restore `throttleLLM(newLLMProviderFromConfig(...))`. Pipeline reverts to single-provider. |
| PR 2 | Remove new config fields. Unused without PR 3.                                             |
| PR 1 | Remove library files. Unused without PR 3.                                                 |

**Emergency env-var overrides (no redeploy for Docker Compose):**

- `LLM_FALLBACK_PROVIDER=` (empty) → chain degrades to primary-only with retry
- `LLM_RETRY_MAX_ATTEMPTS=1` → no retry
- `LLM_BUDGET_REQUESTS_DAY=0` → no budget guard (unlimited)
- `LLM_CALL_TIMEOUT=30m` → even more generous timeout
- `LLM_THROTTLE_CONCURRENCY=1` → serialize all LLM calls

## Risks

| Risk                                                                               | Mitigation                                                                                                                                                          |
| ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Fallback storm: primary down → all traffic hits secondary → secondary rate-limited | Budget guard caps daily requests. Auto-disable after 5 consecutive failures. Alert fires.                                                                           |
| Paid provider auto-charges on overflow                                             | `BudgetGuardProvider` returns `ErrBudgetExhausted` before call. Provider chain stops. Never silent spend.                                                           |
| Report worker blocks pipeline                                                      | Reports are async job. Pipeline reads latest snapshot. Stale report = warning, not failure.                                                                         |
| Migration breaks existing schema                                                   | Migration is additive (new table only). Down migration is `DROP TABLE`. No existing table altered.                                                                  |
| Very high timeouts mask real issues                                                | Intentional per user request during stabilization. Phase 2 can tighten once pipeline is functioning. Metrics track actual latency distribution.                     |
| OpenRouter free model selection non-deterministic                                  | OpenRouter routes to cheapest/available free model. We don't pick model → set `OPENROUTER_MODEL=` empty or use their default routing. Acceptable for fallback-tier. |
| GitHub Models rate limits strict (15 RPM free)                                     | Used as last-resort fallback only. Budget guard prevents burst. Jitter spreads calls.                                                                               |

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    LLM Provider Chain                       │
│                                                             │
│  ┌─────────────┐   ┌─────────────┐   ┌──────────────────┐  │
│  │  Throttle   │──▶│   Retry     │──▶│    Fallback      │  │
│  │  (N conc.)  │   │  (2 att.)   │   │ primary→second.  │  │
│  └─────────────┘   └─────────────┘   └──────────────────┘  │
│         │                                      │            │
│         │                              ┌───────┴──────┐     │
│         │                              │    Cache     │     │
│         │                              │  (in-memory) │     │
│         │                              └──────────────┘     │
│         │                                                   │
│  ┌──────┴──────┐                                            │
│  │   Budget    │  ← ErrBudgetExhausted (not retryable)     │
│  │   Guard     │                                            │
│  └─────────────┘                                            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   Report Worker (automation job)            │
│                                                             │
│  Cron: 0 17 * * 1-5 ET (after market close)                │
│  For each active paper strategy:                            │
│    1. Load metrics from latest runs                         │
│    2. papervalidation.GenerateReport()                      │
│    3. Persist to report_artifacts                           │
│    4. On error: persist error row, continue                 │
│                                                             │
│  API: GET /api/v1/strategies/{id}/reports/latest            │
│       includes stale_seconds field                          │
└─────────────────────────────────────────────────────────────┘
```

## New Environment Variables Summary

| Variable                   | Type     | Default | Description                                       |
| -------------------------- | -------- | ------- | ------------------------------------------------- |
| `LLM_FALLBACK_PROVIDER`    | string   | `""`    | Secondary LLM provider name (empty = no fallback) |
| `LLM_FALLBACK_MODEL`       | string   | `""`    | Model override for fallback provider              |
| `LLM_RETRY_MAX_ATTEMPTS`   | int      | `2`     | Max retry attempts on transient errors            |
| `LLM_CALL_TIMEOUT`         | duration | `5m`    | Per-call timeout (high during stabilization)      |
| `LLM_BUDGET_REQUESTS_DAY`  | int      | `0`     | Max requests/day (0 = unlimited)                  |
| `LLM_BUDGET_TOKENS_DAY`    | int      | `0`     | Max tokens/day (0 = unlimited)                    |
| `LLM_THROTTLE_CONCURRENCY` | int      | `4`     | Max concurrent LLM calls                          |

## Verification Commands

```bash
# After each PR:
go build ./cmd/tradingagent
go test -short -count=1 ./...

# After PR 4 (migration):
task migrate:up
# Verify schema version = 17

# After all PRs:
golangci-lint run ./...
```
