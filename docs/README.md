---
title: "Documentation"
description: "Canonical documentation hub for get-rich-quick."
status: "canonical"
updated: "2026-04-03"
tags: [docs, canonical, index]
---

# Documentation

This directory now has a single job: explain the application as it exists today, separate what is implemented from what is merely planned, and keep older design research available without pretending it is the current runtime contract.

## Start here

| Page | Use it for |
| --- | --- |
| [Getting Started](getting-started.md) | Fastest path from clone to a working local stack, first login, first strategy, and first run |
| [Development Setup](development-setup.md) | Full local development workflow, migrations, testing, frontend setup, and day-to-day commands |
| [Reference](reference/README.md) | Source-of-truth technical reference for API, CLI, architecture, runtime agents, config, UI, and integrations |
| [Runbooks](runbooks/README.md) | Operational procedures for incidents, safety controls, and routine interventions |
| [Known Issues](known-issues.md) | Current gaps, rough edges, and repo-health problems that affect operators and contributors |
| [Roadmap](roadmap.md) | Proposed next steps and the major product/engineering themes that follow from the current codebase |

## What the application does

`get-rich-quick` is a Go-based trading application with three operator surfaces:

1. A REST API and WebSocket server for programmatic control and real-time updates.
2. A React web UI for strategies, runs, portfolio state, memories, settings, risk, and live activity.
3. A Cobra CLI and Bubble Tea dashboard for local control and terminal-first operations.

Underneath those surfaces is a multi-agent trading runtime that:

1. Pulls market data, fundamentals, news, and sentiment where available.
2. Runs analyst agents in parallel.
3. Runs a research debate.
4. Builds a trade plan.
5. Runs a risk debate.
6. Applies hard risk controls.
7. Routes to paper or live execution adapters depending on strategy settings and broker configuration.

## Feature map

### Core platform

- Strategy CRUD with schedules, paper/live mode, skip-next-run support, and typed JSON config validation.
- Manual strategy execution through the API, UI, and CLI.
- Persistent pipeline runs, run snapshots, agent decisions, events, conversations, audit records, orders, positions, trades, and memories.
- JWT login plus API key authentication for protected API routes.
- WebSocket event streaming for live run and system activity.

### Trading runtime

- Parallel analyst phase with market, fundamentals, news, and social roles.
- Bull/bear research debate and research-manager synthesis.
- Trader phase that generates entry, sizing, stop, and target plans.
- Risk debate with aggressive, conservative, and neutral perspectives plus a risk-manager final signal.
- Hard risk engine with kill switch, circuit breaker, exposure caps, and pre-trade checks.

### Market coverage

- `stock` strategies.
- `crypto` strategies.
- `polymarket` strategies as a market type in domain/runtime logic, with incomplete live execution support.

### Integrations

- LLM providers: OpenAI, Anthropic, Google, OpenRouter, xAI, Ollama.
- Market data: Polygon, Alpha Vantage, Yahoo Finance, Binance.
- Brokers/execution: Alpaca, Binance, local paper broker, Polymarket adapter package.
- Notifications: Telegram, email/SMTP, Discord webhooks, PagerDuty webhooks, n8n webhooks.
- Ops/infra: PostgreSQL, Redis, Docker Compose, Prometheus, Grafana.

## Canonical vs archive

The pages below are the current source of truth for how the app actually behaves:

- [Getting Started](getting-started.md)
- [Development Setup](development-setup.md)
- [Reference](reference/README.md)
- [Runbooks](runbooks/README.md)
- [Known Issues](known-issues.md)
- [Roadmap](roadmap.md)

The rest of `docs/` is still valuable, but it should be read with context:

- [ADRs](adr/README.md) record major decisions and their rationale.
- [Research](research/index.md) captures strategy, execution, and LLM research that informed the product.
- `docs/design/` contains design/spec material. Some pages still describe intended architecture rather than the exact runtime wiring that exists today.
- Historical planning documents such as `phase-*-execution-paths.md`, `implementation-board.md`, and audit notes remain as archive material.

## Reading order

If you are new to the project:

1. Read [Getting Started](getting-started.md).
2. Read [Development Setup](development-setup.md) if you plan to contribute.
3. Read [Reference](reference/README.md) to understand the real implementation surface.
4. Read [Known Issues](known-issues.md) before assuming every described feature is production-ready.

If you are operating the system:

1. Read [Runbooks](runbooks/README.md).
2. Review [Reference: API](reference/api.md), [Reference: Web UI](reference/web-ui.md), and [Reference: Configuration](reference/configuration.md).
3. Keep [Known Issues](known-issues.md) open when debugging surprising behavior.

If you are planning future work:

1. Read [Roadmap](roadmap.md).
2. Review the [ADRs](adr/README.md).
3. Use [Research](research/index.md) for background, not for implementation truth.
