---
title: "Strategy Config"
description: "Typed strategy configuration schema, defaults, validation, and examples."
status: "canonical"
updated: "2026-04-03"
tags: [strategy, config, reference]
---

# Strategy Config

Strategies persist a free-form JSON `config` field, but the API validates that JSON against the typed config model in `internal/agent/strategy_config.go`.

## Where it is used

- persisted on `domain.Strategy.Config`
- validated by the API on create/update
- resolved into concrete runtime values before a run

## Top-level shape

```json
{
  "llm_config": {},
  "pipeline_config": {},
  "risk_config": {},
  "analyst_selection": [],
  "prompt_overrides": {}
}
```

All sections are optional.

## `llm_config`

```json
{
  "provider": "openai",
  "deep_think_model": "gpt-5.2",
  "quick_think_model": "gpt-5-mini"
}
```

Fields:

| Field | Type | Meaning |
| --- | --- | --- |
| `provider` | string | one of `openai`, `anthropic`, `google`, `openrouter`, `xai`, `ollama` |
| `deep_think_model` | string | model for deeper reasoning phases |
| `quick_think_model` | string | model for faster analyst phases |

Notes:

- some providers have an explicit model allowlist
- `openrouter`, `xai`, and `ollama` are validated more loosely at the provider/model pairing level

## `pipeline_config`

```json
{
  "debate_rounds": 3,
  "analysis_timeout_seconds": 30,
  "debate_timeout_seconds": 60
}
```

Fields:

| Field | Type | Validation |
| --- | --- | --- |
| `debate_rounds` | integer | must be `>= 1` |
| `analysis_timeout_seconds` | integer | must be `>= 1` |
| `debate_timeout_seconds` | integer | must be `>= 1` |

## `risk_config`

```json
{
  "position_size_pct": 5,
  "stop_loss_multiplier": 1.5,
  "take_profit_multiplier": 2.0,
  "min_confidence": 0.65
}
```

Fields:

| Field | Type | Validation |
| --- | --- | --- |
| `position_size_pct` | number | between `0` and `100` |
| `stop_loss_multiplier` | number | must be `> 0` |
| `take_profit_multiplier` | number | must be `> 0` |
| `min_confidence` | number | between `0` and `1` |

Behavioral notes:

- `position_size_pct` feeds order sizing
- `min_confidence` can downgrade a trade to `hold`
- stop-loss and take-profit multipliers influence the generated plan

## `analyst_selection`

Example:

```json
{
  "analyst_selection": ["market_analyst", "news_analyst"]
}
```

Semantics:

- `nil` or omitted means all analysts are enabled
- a provided list restricts runtime execution to those roles

Use this to trade off:

- cost
- latency
- signal diversity

## `prompt_overrides`

Example:

```json
{
  "prompt_overrides": {
    "market_analyst": "Focus on short-term momentum and gap continuation."
  }
}
```

Keys must be valid agent roles.

## Default resolution

If a field is omitted, the runtime resolves it through:

1. strategy config
2. global settings surface
3. hardcoded defaults

Current hardcoded defaults:

| Field | Default |
| --- | --- |
| `provider` | `openai` |
| `deep_think_model` | `gpt-5.2` |
| `quick_think_model` | `gpt-5-mini` |
| `debate_rounds` | `3` |
| `analysis_timeout_seconds` | `30` |
| `debate_timeout_seconds` | `60` |
| `position_size_pct` | `5.0` |
| `stop_loss_multiplier` | `1.5` |
| `take_profit_multiplier` | `2.0` |
| `min_confidence` | `0.65` |

## Minimal example

```json
{
  "risk_config": {
    "position_size_pct": 5
  }
}
```

## Practical stock example

```json
{
  "llm_config": {
    "provider": "openai",
    "deep_think_model": "gpt-5.2",
    "quick_think_model": "gpt-5-mini"
  },
  "pipeline_config": {
    "debate_rounds": 2,
    "analysis_timeout_seconds": 45,
    "debate_timeout_seconds": 45
  },
  "risk_config": {
    "position_size_pct": 4,
    "stop_loss_multiplier": 1.25,
    "take_profit_multiplier": 2.5,
    "min_confidence": 0.7
  },
  "analyst_selection": [
    "market_analyst",
    "fundamentals_analyst",
    "news_analyst"
  ]
}
```

## Practical crypto example

```json
{
  "llm_config": {
    "provider": "openrouter",
    "deep_think_model": "openai/gpt-4.1-mini",
    "quick_think_model": "gpt-5-mini"
  },
  "risk_config": {
    "position_size_pct": 2,
    "min_confidence": 0.75
  },
  "analyst_selection": [
    "market_analyst",
    "news_analyst"
  ]
}
```

## Validation failure examples

These are rejected:

- unknown provider
- unknown model
- provider/model mismatch for constrained providers
- negative timeout values
- `position_size_pct > 100`
- `min_confidence > 1`
- unknown roles in `analyst_selection`
- unknown roles in `prompt_overrides`

## UI/API implications

- the API is the authoritative validator
- the current structured strategy editor in the UI needs revalidation because related frontend files currently contain merge conflicts
- if the UI behaves strangely, validate the payload against the API directly
