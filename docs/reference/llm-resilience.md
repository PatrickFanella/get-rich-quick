---
title: 'LLM Resilience'
description: 'Provider chain architecture, env var reference, and operational troubleshooting.'
status: 'canonical'
created: '2026-04-16'
tags: [llm, resilience, observability, reference]
---

## Provider Chain Architecture

Every LLM call passes through a layered provider chain. Each layer is optional — omitting its config disables it.

```text
┌─────────────────────────────────────────────────────────────┐
│                    LLM Provider Chain                       │
│                                                             │
│  Request flow (outermost → innermost):                      │
│                                                             │
│  ┌──────────────┐                                           │
│  │ Budget Guard  │  ErrBudgetExhausted → not retried        │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │   Timeout    │  Per-call deadline (LLM_CALL_TIMEOUT)     │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │  Throttle    │  Concurrency limiter (LLM_THROTTLE_…)     │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │    Retry     │  Exponential backoff on 429/5xx           │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │  Fallback    │  Primary fails → try secondary            │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │    Cache     │  In-memory response cache                 │
│  └──────┬───────┘                                           │
│         ▼                                                   │
│  ┌──────────────┐                                           │
│  │   Provider   │  Raw OpenAI / OpenRouter / Ollama / etc.  │
│  └──────────────┘                                           │
└─────────────────────────────────────────────────────────────┘
```

### Chain construction

`NewProviderChain(primary, logger, opts...)` builds layers inside-out:

1. **Cache** (innermost) — deduplicates identical prompts
2. **Fallback** — routes to secondary provider on failure
3. **Retry** — exponential backoff with jitter on transient errors
4. **Throttle** — semaphore-based concurrency limiter
5. **Timeout** — per-call `context.WithTimeout`
6. **Budget Guard** (outermost) — rejects when daily limits hit

Only layers whose options are provided are added. With zero config the chain degrades to the raw provider.

## Environment Variables

| Variable                   | Type     | Default  | Description                                             |
| -------------------------- | -------- | -------- | ------------------------------------------------------- |
| `LLM_DEFAULT_PROVIDER`     | string   | `openai` | Primary LLM provider name                               |
| `LLM_FALLBACK_PROVIDER`    | string   | `""`     | Secondary LLM provider (empty = no fallback)            |
| `LLM_FALLBACK_MODEL`       | string   | `""`     | Model override for fallback provider                    |
| `LLM_RETRY_MAX_ATTEMPTS`   | int      | `2`      | Max retry attempts on transient errors (429, 5xx)       |
| `LLM_CALL_TIMEOUT`         | duration | `5m`     | Per-call timeout (high during stabilization)            |
| `LLM_BUDGET_REQUESTS_DAY`  | int      | `0`      | Max requests/day (0 = unlimited)                        |
| `LLM_BUDGET_TOKENS_DAY`    | int      | `0`      | Max tokens/day (0 = unlimited)                          |
| `LLM_THROTTLE_CONCURRENCY` | int      | `4`      | Max concurrent LLM calls                                |
| `LLM_CACHE_ENABLED`        | bool     | `true`   | Enable in-memory response cache                         |
| `LLM_TIMEOUT`              | duration | `30s`    | Legacy timeout (superseded by `LLM_CALL_TIMEOUT`)       |
| `LLM_DEBATE_TIMEOUT`       | duration | —        | Debate-phase-specific timeout with quick-model fallback |
| `SCHEDULER_JOB_TIMEOUT`    | duration | `45m`    | Pipeline-level hard cap                                 |

### Emergency env-var overrides (no redeploy needed for Docker Compose)

| Override                     | Effect                                    |
| ---------------------------- | ----------------------------------------- |
| `LLM_FALLBACK_PROVIDER=`     | Chain degrades to primary-only with retry |
| `LLM_RETRY_MAX_ATTEMPTS=1`   | Disables retry                            |
| `LLM_BUDGET_REQUESTS_DAY=0`  | Disables budget guard (unlimited)         |
| `LLM_CALL_TIMEOUT=30m`       | Even more generous timeout                |
| `LLM_THROTTLE_CONCURRENCY=1` | Serialize all LLM calls                   |

## Report Worker

The `paper_validation_report` automation job generates paper-trading validation reports.

- **Schedule:** `0 17 * * 1-5` ET (after US market close)
- **Per-strategy jitter:** 0–119s to spread LLM/DB load
- **Persistence:** `report_artifacts` table (upsert on `strategy_id, report_type, time_bucket`)
- **Error handling:** Each strategy is independent — a failure in one does not block others

### Auto-Disable

If a job accumulates **5 consecutive failures**, it is automatically disabled:

- Job's `Enabled` flag is set to `false`
- A log entry at ERROR level is emitted
- The job will not fire again until manually re-enabled

**Re-enable via API:**

```http
PUT /api/v1/automation/jobs/{name}/enable
```

### API Endpoints

| Endpoint                                 | Method | Description                         |
| ---------------------------------------- | ------ | ----------------------------------- |
| `/api/v1/strategies/{id}/reports/latest` | GET    | Latest completed report + staleness |
| `/api/v1/strategies/{id}/reports`        | GET    | Paginated report list               |

The `latest` endpoint includes a `stale_seconds` field = `now - completed_at`.

## Prometheus Metrics

### LLM Provider Chain

| Metric                                    | Type      | Labels                | Description                             |
| ----------------------------------------- | --------- | --------------------- | --------------------------------------- |
| `tradingagent_llm_calls_total`            | Counter   | provider, model, role | Total LLM API calls                     |
| `tradingagent_llm_retry_total`            | Counter   | provider              | Retry attempts                          |
| `tradingagent_llm_fallback_total`         | Counter   | reason                | Fallback events                         |
| `tradingagent_llm_budget_exhausted_total` | Counter   | —                     | Budget exhaustion rejections            |
| `tradingagent_llm_tokens_total`           | Counter   | type                  | Token consumption (prompt / completion) |
| `tradingagent_llm_latency_seconds`        | Histogram | provider, model       | Call latency distribution               |
| `tradingagent_llm_cache_hits_total`       | Counter   | —                     | Response cache hits                     |
| `tradingagent_llm_cache_misses_total`     | Counter   | —                     | Response cache misses                   |

### Report Worker

| Metric                                     | Type      | Labels      | Description                   |
| ------------------------------------------ | --------- | ----------- | ----------------------------- |
| `tradingagent_report_worker_success_total` | Counter   | strategy_id | Successful report generations |
| `tradingagent_report_worker_error_total`   | Counter   | strategy_id | Failed report generations     |
| `tradingagent_report_staleness_seconds`    | Histogram | strategy_id | Report age at query time      |
| `tradingagent_automation_job_errors_total` | Counter   | job_name    | All automation job errors     |

## Troubleshooting

### All providers down

**Symptoms:** `tradingagent_llm_fallback_total` spikes, pipeline runs fail.

**Check:**

1. `tradingagent_llm_calls_total` by provider — are calls reaching providers?
2. Provider status pages (OpenAI, OpenRouter)
3. API key validity

**Mitigate:** If secondary is also down, set `LLM_FALLBACK_PROVIDER=` to avoid wasted fallback attempts.

### Budget exhausted

**Symptoms:** `tradingagent_llm_budget_exhausted_total` incrementing, jobs returning `ErrBudgetExhausted`.

**Check:**

1. `GET /api/v1/llm/budget` (if exposed) or check logs for budget stats
2. Compare `LLM_BUDGET_REQUESTS_DAY` against actual daily volume

**Mitigate:** Set `LLM_BUDGET_REQUESTS_DAY=0` to remove limit temporarily. Investigate cost after.

### Report stale

**Symptoms:** `tradingagent_report_staleness_seconds` > 86400 (24h), dashboard shows stale warning.

**Check:**

1. `GET /api/v1/automation/status` — is `paper_validation_report` enabled? Check `consecutive_failures`.
2. If auto-disabled: re-enable via `PUT /api/v1/automation/jobs/paper_validation_report/enable`
3. Check logs for the underlying failure reason

**Mitigate:** Trigger manual run via `POST /api/v1/automation/jobs/paper_validation_report/run`.

### Fallback storm

**Symptoms:** `tradingagent_llm_fallback_total` / `tradingagent_llm_calls_total` > 20% over sustained period.

**Check:**

1. Primary provider health
2. `tradingagent_llm_retry_total` — are retries exhausting before fallback?

**Mitigate:** Budget guard caps daily requests. Auto-disable fires after 5 consecutive job failures. Reduce `LLM_THROTTLE_CONCURRENCY=1` to slow the bleed.

## Success Criteria (7-Day Validation)

| Criterion                  | Target   |
| -------------------------- | -------- |
| Report worker success rate | > 98%    |
| Fallback usage             | < 20%    |
| Pipeline timeout failures  | ↓ 80%    |
| P95 report latency         | < 20 min |
