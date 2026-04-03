---
title: "Data Providers"
description: "Market data provider chains, caching, historical downloads, and current gaps."
status: "canonical"
updated: "2026-04-03"
tags: [data, providers, reference]
---

# Data Providers

The market-data layer is defined in `internal/data`.

## Abstraction

Every provider implements:

- OHLCV retrieval
- fundamentals retrieval
- news retrieval
- social sentiment retrieval

The abstraction is broad on purpose. In practice, provider support is uneven, and the chain falls back when a provider does not support a requested method.

## Provider chain model

The app constructs separate chains by market type.

### Stock chain

Current order:

1. Polygon when configured
2. Alpha Vantage when configured
3. Yahoo Finance as a final fallback

### Crypto chain

Current order:

1. Binance

### Unsupported or partial surfaces

- `FINNHUB_*` config exists but is not currently wired into the provider factory
- a `newsapi` package exists but is not part of the main runtime chain
- social sentiment support depends on provider implementation and is not universally available

## Caching

`DataService` caches retrieved data through the market-data cache repository.

Current cache buckets include:

- OHLCV
- fundamentals
- news
- social sentiment

Representative TTL behavior:

- OHLCV uses timeframe-sensitive caching
- fundamentals are cached for longer windows such as hours
- news is cached for shorter windows such as minutes

## Historical OHLCV downloads

`DataService` also supports bulk historical downloads into the historical OHLCV repository.

Use cases:

- backtesting
- warm caches
- preloading historical ranges

Features:

- market-type-aware chain selection
- incremental gap detection
- upsert into persisted OHLCV storage

## What the runtime actually uses

The production strategy runner loads initial state from the data service and may consume:

- price bars and computed indicators
- fundamentals
- news articles
- latest social snapshot

If a provider chain cannot satisfy a surface, the runner may continue with a partial seed.

## Coverage summary

| Surface | Stock | Crypto | Notes |
| --- | --- | --- | --- |
| OHLCV | yes | yes | strongest coverage |
| Fundamentals | mostly stock-oriented | limited | depends on provider implementation |
| News | partial | partial | package support exists, runtime wiring is uneven |
| Social sentiment | partial | partial | interface exists, live support is incomplete |

## Operational implications

- provider availability directly affects run quality
- a strategy can still run with incomplete analyst context
- a configured env var does not guarantee the provider is actually instantiated
- when debugging a “missing data” issue, inspect the chain wiring, not just `.env`
