---
title: "Agents and Runtime"
description: "How the multi-agent trading runtime is assembled and executed in the current application."
status: "canonical"
updated: "2026-04-03"
tags: [agents, runtime, reference]
---

# Agents and Runtime

This document explains how the current strategy runner actually behaves.

## Runtime entry points

The important files are:

- `cmd/tradingagent/runtime.go`
- `cmd/tradingagent/prod_strategy_runner.go`
- `internal/agent/resolve_config.go`
- `internal/agent/runner.go`
- `internal/agent/strategy_config.go`

## Runtime model

A strategy run uses:

1. a persisted `domain.Strategy`
2. a typed `agent.StrategyConfig` parsed from the strategy’s JSON `config`
3. system-level defaults from the settings/bootstrap layer
4. hardcoded fallbacks from `ResolveConfig`

The resulting run is then executed by the production strategy runner.

## Agent roster

### Analysis phase

- `market_analyst`
- `fundamentals_analyst`
- `news_analyst`
- `social_media_analyst`

### Research debate phase

- `bull_researcher`
- `bear_researcher`
- `research_manager` / invest judge role

### Trading phase

- `trader`

### Risk debate phase

- `aggressive_analyst`
- `conservative_analyst`
- `neutral_analyst`
- `risk_manager`

## Phase flow

The runtime follows these conceptual phases:

```text
analysis -> research_debate -> trading -> risk_debate
```

### 1. Analysis

The runner seeds market context, then executes the enabled analysts in parallel.

Inputs may include:

- OHLCV bars and indicators
- fundamentals
- recent news
- social sentiment snapshots

### 2. Research debate

Bull and bear roles argue over the evidence. A manager/judge role synthesizes the result into an investment plan.

### 3. Trading

The trader converts the investment plan into a concrete trade plan:

- side
- sizing
- stop-loss framing
- take-profit framing
- execution details

### 4. Risk debate

The aggressive, conservative, and neutral roles debate the trade plan. The risk manager emits the final actionable signal such as `buy`, `sell`, or `hold` plus confidence.

### 5. Hard risk and execution

The signal does not directly guarantee execution. The hard risk engine and order-management layer still apply:

- kill switch
- circuit breaker
- position limits
- exposure caps
- live vs paper execution routing

## Config resolution

`agent.ResolveConfig` merges values in this order:

1. strategy config overrides
2. global settings
3. hardcoded defaults

Current hardcoded defaults:

| Field | Default |
| --- | --- |
| provider | `openai` |
| deep think model | `gpt-5.2` |
| quick think model | `gpt-5-mini` |
| debate rounds | `3` |
| analysis timeout | `30` seconds |
| debate timeout | `60` seconds |
| position size | `5.0` percent |
| stop-loss multiplier | `1.5` |
| take-profit multiplier | `2.0` |
| minimum confidence | `0.65` |

## Analyst selection

Strategies can restrict which analyst roles run via `analyst_selection`.

Semantics:

- `nil` analyst selection means all analysts are enabled
- a non-empty list means only those roles run

This is one of the most useful ways to tune cost and latency per strategy.

## Prompt overrides

Strategies can replace the default system prompt for individual roles via `prompt_overrides`.

Practical caution:

- prompt overrides are powerful but easy to abuse
- treat them as precision overrides, not a place to dump strategy prose

## LLM provider routing

The runtime supports these provider names:

- `openai`
- `anthropic`
- `google`
- `openrouter`
- `xai`
- `ollama`

Provider wiring details:

- `openrouter` and `xai` are treated as OpenAI-compatible transports in the production runner
- the strategy config validator enforces provider/model allowlists for some providers and looser validation for others

## Initial state seeding

Before agent execution, the production runner loads initial state from the data service.

That seed can include:

- market bars plus indicators
- fundamentals
- news articles
- latest social sentiment snapshot

The completeness of the seed depends on provider availability for the selected market type.

## Timeouts

The runtime uses phase-specific timeouts from resolved config.

Important caveat:

- the current runtime does not enforce a meaningful whole-pipeline timeout; only phase-level limits are effectively applied

## Execution routing

The production strategy runner chooses brokers roughly as follows:

- paper stock:
  - Alpaca paper when configured and allowed
  - otherwise local paper broker
- paper crypto:
  - Binance paper/testnet when configured and allowed
  - otherwise local paper broker
- live stock:
  - Alpaca when live trading is enabled and configured
- live crypto:
  - Binance when live trading is enabled and configured
- live polymarket:
  - not yet presented as a fully supported path in the main runner

## Persistence

A run can produce and persist:

- pipeline run records
- agent decisions
- events
- snapshots
- orders
- positions
- trades
- memories
- audit records

## Runtime caveats

- The repo still contains unresolved merge conflicts in some runtime-related files.
- The newer runtime path is the production truth; older pipeline abstractions still exist and are useful context, but they are not the whole story.
- Settings-driven config changes are not durable across restart.
