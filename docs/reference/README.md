---
title: "Reference"
description: "Implementation reference for the current get-rich-quick application."
status: "canonical"
updated: "2026-04-03"
tags: [reference, index]
---

# Reference

These pages are the source-of-truth reference set for the application as it is wired today.

## Core reference

| Page | Covers |
| --- | --- |
| [Architecture](architecture.md) | Package layout, runtime flow, major interfaces, persistence, and event flow |
| [API](api.md) | REST routes, auth model, WebSocket protocol, response shapes, and route-group behavior |
| [CLI](cli.md) | Cobra commands, flags, environment variables, and terminal dashboard entry points |
| [Web UI](web-ui.md) | Frontend route map, page-by-page behavior, and current limitations |
| [Configuration](configuration.md) | Environment variables, runtime settings, secrets, feature flags, and what persists |

## Trading/runtime reference

| Page | Covers |
| --- | --- |
| [Agents and Runtime](agents.md) | Agent roster, phase sequencing, config resolution, runtime wiring, and execution path |
| [Strategy Config](strategy-config.md) | Typed JSON config structure, validation, defaults, and examples |
| [Data Providers](data-providers.md) | Provider chains, caching, market coverage, historical downloads, and gaps |
| [LLM Providers](llm.md) | Provider support, model routing, defaults, and current runtime behavior |

## Supporting docs

- [Known Issues](../known-issues.md) for current implementation gaps
- [Runbooks](../runbooks/README.md) for operator procedures
- [Roadmap](../roadmap.md) for proposed next work

## Scope rules

When this reference disagrees with older pages in `docs/design/` or archived planning notes, prefer this reference. Those other pages are still useful context, but they are not treated as the current runtime contract.
