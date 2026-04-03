---
title: "Runbooks"
description: "Operational procedures for common incidents, controls, and interventions in get-rich-quick."
status: "canonical"
updated: "2026-04-03"
tags: [runbooks, operations]
---

# Runbooks

These runbooks are for operators and contributors handling a running system, an incident, or a safety intervention.

## Before you start

- Export `TRADINGAGENT_API_URL` if the API is not on `http://127.0.0.1:8080`.
- Export either `TRADINGAGENT_TOKEN` or `TRADINGAGENT_API_KEY` for authenticated CLI/API calls.
- Assume the system may already be in a degraded state; gather evidence before changing it.
- Read [Known Issues](../known-issues.md) if behavior seems inconsistent with older docs.

## Safety-first runbooks

- [Emergency kill switch activation](emergency-kill-switch.md)
- [Circuit breaker investigation and reset](circuit-breaker.md)
- [Investigating a bad trade](bad-trade.md)
- [Reviewing agent decisions for a run](review-agent-decisions.md)

## Platform and dependency runbooks

- [Broker API outage handling](broker-api-outage.md)
- [LLM provider outage handling](llm-provider-outage.md)
- [Rolling restart procedure](rolling-restart.md)
- [Database backup and restore](database-backup-restore.md)

## Routine operator tasks

- [Adding a new strategy](add-strategy.md)

## Notes on scope

These runbooks assume the current implementation reality:

- settings edits are not durable across restart
- WebSocket access is not yet treated as a hardened public surface
- some runtime/frontend areas still need cleanup because of unresolved merge conflicts elsewhere in the repo
