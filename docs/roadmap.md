---
title: "Roadmap"
description: "Planned and proposed future work inferred from the current codebase and existing design material."
status: "canonical"
updated: "2026-04-03"
tags: [roadmap, planning]
---

# Roadmap

This page is deliberately split between what the code already implies and what is still only a proposal. Nothing here should be read as a committed delivery date.

## Guiding priorities

The highest-leverage future work is not adding yet another speculative feature. It is closing the gap between the ambitious design and the uneven current implementation.

## Near-term priorities

### 1. Stabilize the repository

- Resolve the merge-conflict markers currently present in backend and frontend files.
- Restore reliable local builds and broad automated test coverage.
- Re-verify the strategy runner, risk engine, and realtime UI after conflict resolution.

### 2. Close security and safety gaps

- Put authentication or signed access in front of the WebSocket endpoint.
- Persist kill-switch and critical safety state more explicitly across restarts.
- tighten auditability around operator actions, manual runs, and settings changes.

### 3. Finish settings and control-plane persistence

- Replace the in-memory settings service with a persisted configuration model.
- define precedence between environment config, persisted settings, and per-strategy overrides.
- Make broker/provider configuration edits survive restarts.

### 4. Turn partial features into supported features

- finish the realtime page and its backing UX flows once merge conflicts are resolved
- expose backtest capabilities through supported API/UI routes
- clarify or complete Polymarket execution behavior
- wire remaining data-provider surfaces that are already represented in config

## Medium-term product work

### Backtesting and validation

- first-class backtest APIs and UI pages
- comparison views between paper/live/backtest outcomes
- richer performance analytics, attribution, and drawdown analysis

### Better operator experience

- richer strategy templates and guided config creation
- better run replays and agent-decision visualizations
- clearer portfolio and risk explanations for non-authors of the system

### Execution hardening

- more explicit live-trading guardrails
- better order-state reconciliation with upstream brokers
- stronger handling of partial fills, retries, and exchange outages

## Longer-term platform work

### Multi-strategy portfolio intelligence

- correlation-aware allocation across strategies
- exposure budgeting across market types
- portfolio-level risk policies that feed back into strategy execution

### Memory and learning improvements

- better retrieval and summarization across historical runs
- memory quality controls so noisy runs do not pollute future prompts
- learning loops that are measurable instead of just aspirational

### Deployment maturity

- explicit production topology guidance beyond local Compose
- recovery-tested backup/restore for stateful services
- stronger observability and alerting defaults

## What is intentionally not treated as done

Even though the repository contains code, docs, or type surfaces for these areas, they are not yet presented as fully finished product capabilities:

- durable settings persistence
- authenticated WebSocket access
- end-user account management/registration
- full backtest product surface
- complete social/news provider coverage across all market types
- fully settled realtime UI implementation

## How to use this roadmap

- If you are contributing code, pair this page with [Known Issues](known-issues.md).
- If you are planning a feature, validate it against [Reference](reference/README.md) first.
- If you are looking for historical rationale, use [ADRs](adr/README.md) and `docs/design/`.
