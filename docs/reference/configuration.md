---
title: "Configuration"
description: "Environment variables, feature flags, runtime settings, and persistence semantics."
status: "canonical"
updated: "2026-04-03"
tags: [configuration, env, reference]
---

# Configuration

All durable server configuration currently starts in the environment.

## Loading rules

- `internal/config.Load()` reads environment variables
- `.env` is auto-loaded only when `APP_ENV=development`
- validation runs after loading
- the API settings surface is separate and currently memory-backed

## Major configuration groups

### Server and auth

- `APP_ENV`
- `APP_HOST`
- `APP_PORT`
- `LOG_LEVEL`
- `JWT_SECRET`

### Database and cache

- `DATABASE_URL`
- `DATABASE_POOL_SIZE`
- `DATABASE_SSL_MODE`
- `REDIS_URL`

### LLM routing

- `LLM_DEFAULT_PROVIDER`
- `LLM_DEEP_THINK_MODEL`
- `LLM_QUICK_THINK_MODEL`
- `LLM_TIMEOUT`
- provider-specific keys and base URLs

### Data providers

- `POLYGON_API_KEY`
- `ALPHA_VANTAGE_API_KEY`
- `ALPHA_VANTAGE_RATE_LIMIT_PER_MINUTE`
- `FINNHUB_API_KEY`
- `FINNHUB_RATE_LIMIT_PER_MINUTE`

### Brokers

- `ALPACA_API_KEY`
- `ALPACA_API_SECRET`
- `ALPACA_PAPER_MODE`
- `BINANCE_API_KEY`
- `BINANCE_API_SECRET`
- `BINANCE_PAPER_MODE`

### Notifications

- Telegram
- SMTP/email
- n8n webhooks
- PagerDuty webhooks
- Discord webhooks

### Risk defaults

- `RISK_MAX_POSITION_SIZE_PCT`
- `RISK_MAX_DAILY_LOSS_PCT`
- `RISK_MAX_DRAWDOWN_PCT`
- `RISK_MAX_OPEN_POSITIONS`
- `RISK_CIRCUIT_BREAKER_THRESHOLD`
- `RISK_CIRCUIT_BREAKER_COOLDOWN`
- `TRADING_AGENT_KILL`

### Feature flags

- `ENABLE_SCHEDULER`
- `ENABLE_REDIS_CACHE`
- `ENABLE_AGENT_MEMORY`
- `ENABLE_LIVE_TRADING`

## What persists vs what does not

### Durable

- environment variables
- database records
- migration state
- strategies and run artifacts

### Not durable today

- settings page edits
- API `PUT /settings` edits

Those settings mutate the in-memory settings service for the running process only.

## Kill switch sources

The risk engine can be influenced by:

- API toggles
- environment variable `TRADING_AGENT_KILL`
- file flag under `/tmp/tradingagent_kill`

This is useful operationally, but it also means you should check more than one source when debugging a stuck kill-switch state.

## Broker visibility in settings

The settings response currently reports broker connection summaries for:

- Alpaca
- Binance

This is a UI/control-plane summary, not proof that every execution path is production-ready.

## Recommended local minimum

```dotenv
APP_ENV=development
JWT_SECRET=replace-this-with-a-real-secret
DATABASE_URL=postgres://postgres:postgres@localhost:5432/tradingagent?sslmode=disable
REDIS_URL=redis://localhost:6379/0
OPENAI_API_KEY=...
```

## Recommended local Ollama minimum

```dotenv
APP_ENV=development
JWT_SECRET=replace-this-with-a-real-secret
DATABASE_URL=postgres://postgres:postgres@localhost:5432/tradingagent?sslmode=disable
REDIS_URL=redis://localhost:6379/0
LLM_DEFAULT_PROVIDER=ollama
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2
```

## Operational cautions

- A configured key in `.env` does not guarantee the runtime currently instantiates that integration.
- The UI can make settings feel persistent even though they are not.
- Secrets entered through the UI are not a substitute for durable secret management.
