---
title: "API Reference"
description: "REST and WebSocket reference for the current get-rich-quick API server."
status: "canonical"
updated: "2026-04-03"
tags: [api, rest, websocket, reference]
---

# API Reference

Canonical route sources:

- `internal/api/server.go`
- `internal/api/handlers.go`
- `internal/api/auth.go`
- `internal/api/websocket.go`
- `internal/api/settings.go`

## Base URLs

```text
REST API:   http://localhost:8080/api/v1
WebSocket:  ws://localhost:8080/ws
Ops:        http://localhost:8080/healthz
            http://localhost:8080/health
            http://localhost:8080/metrics
```

## Authentication model

### Public endpoints

- `GET /healthz`
- `GET /health`
- `GET /metrics`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /ws`

### Protected endpoints

Everything else under `/api/v1/*` requires one of:

```http
Authorization: Bearer <access_token>
```

or

```http
X-API-Key: <api_key>
```

### Auth notes

- JWT access and refresh tokens are minted by `AuthManager`.
- API keys are supported through `X-API-Key`.
- API keys are subject to a token-bucket limiter.
- WebSocket traffic is currently not behind the auth middleware. Treat it as an internal/local surface until that changes.

## Common response shapes

### Error envelope

```json
{
  "error": "authentication required",
  "code": "ERR_UNAUTHORIZED"
}
```

### List envelope

```json
{
  "data": [],
  "limit": 50,
  "offset": 0
}
```

Notes:

- `limit` defaults to `50`
- `limit` is capped at `100`
- handlers generally omit `total` even though a shared list type can represent it

## Route map

| Group | Routes |
| --- | --- |
| Auth | `POST /auth/login`, `POST /auth/refresh` |
| Strategies | `GET/POST /strategies`, `GET/PUT/DELETE /strategies/{id}`, lifecycle actions under `/run`, `/pause`, `/resume`, `/skip-next` |
| Runs | `GET /runs`, `GET /runs/{id}`, `GET /runs/{id}/decisions`, `POST /runs/{id}/cancel`, `GET /runs/{id}/snapshot` |
| Portfolio | `GET /portfolio/positions`, `GET /portfolio/positions/open`, `GET /portfolio/summary` |
| Orders | `GET /orders`, `GET /orders/{id}` |
| Trades | `GET /trades` |
| Memories | `GET /memories`, `POST /memories/search`, `DELETE /memories/{id}` |
| Risk | `GET /risk/status`, `POST /risk/killswitch` |
| Settings | `GET /settings`, `PUT /settings` |
| Events | `GET /events` |
| Conversations | `GET/POST /conversations`, `GET/POST /conversations/{id}/messages` |
| Audit log | `GET /audit-log` |

## Endpoint reference

### Auth

#### `POST /api/v1/auth/login`

- auth: public
- body: `username`, `password`
- behavior: loads the user by username, verifies bcrypt password, and returns access plus refresh tokens

Example:

```json
{
  "username": "demo",
  "password": "demo-pass"
}
```

Response:

```json
{
  "access_token": "eyJhbGciOiJI...",
  "refresh_token": "eyJhbGciOiJI...",
  "expires_at": "2026-04-03T18:40:00Z"
}
```

#### `POST /api/v1/auth/refresh`

- auth: public
- body: `refresh_token`
- behavior: validates the refresh token and returns a new pair

### Strategies

#### `GET /api/v1/strategies`

- auth: required
- filters:
  - `ticker`
  - `market_type`
  - `status`
  - `is_paper`
  - `limit`
  - `offset`

#### `POST /api/v1/strategies`

- auth: required
- body: `domain.Strategy`
- validation:
  - required `name`
  - required `ticker`
  - valid `market_type`
  - valid `status`
  - valid typed JSON config if `config` is present

#### `GET /api/v1/strategies/{id}`

- auth: required
- returns one strategy

#### `PUT /api/v1/strategies/{id}`

- auth: required
- full update semantics on the strategy record

#### `DELETE /api/v1/strategies/{id}`

- auth: required
- deletes the strategy record

#### `POST /api/v1/strategies/{id}/run`

- auth: required
- behavior:
  - loads the strategy
  - invokes the configured `StrategyRunner`
  - persists and returns the run result
  - broadcasts run, signal, order, and position events over the hub

#### `POST /api/v1/strategies/{id}/pause`

- auth: required
- behavior: updates strategy status to `paused`

#### `POST /api/v1/strategies/{id}/resume`

- auth: required
- behavior: updates strategy status back to `active`

#### `POST /api/v1/strategies/{id}/skip-next`

- auth: required
- behavior: toggles skip-next-run state for scheduled execution

### Runs

#### `GET /api/v1/runs`

- auth: required
- supports pagination and filters from the runs handlers
- used heavily by the web UI run history page

#### `GET /api/v1/runs/{id}`

- auth: required
- returns the pipeline run summary record

#### `GET /api/v1/runs/{id}/decisions`

- auth: required
- returns agent decisions associated with the run

#### `POST /api/v1/runs/{id}/cancel`

- auth: required
- cancellation surface for in-flight runs where supported

#### `GET /api/v1/runs/{id}/snapshot`

- auth: required
- returns persisted run snapshot data for richer inspection/replay

### Portfolio

#### `GET /api/v1/portfolio/positions`

- auth: required
- returns positions

#### `GET /api/v1/portfolio/positions/open`

- auth: required
- returns current open positions

#### `GET /api/v1/portfolio/summary`

- auth: required
- returns portfolio summary figures used by the dashboard and CLI

### Orders and trades

#### `GET /api/v1/orders`

- auth: required
- list orders

#### `GET /api/v1/orders/{id}`

- auth: required
- get one order

#### `GET /api/v1/trades`

- auth: required
- list trades

### Memories

#### `GET /api/v1/memories`

- auth: required
- list stored agent memories

#### `POST /api/v1/memories/search`

- auth: required
- body includes a natural-language `query`
- returns the most relevant memories first

#### `DELETE /api/v1/memories/{id}`

- auth: required
- deletes one memory record

### Risk

#### `GET /api/v1/risk/status`

- auth: required
- returns:
  - high-level risk status
  - circuit-breaker state
  - kill-switch state
  - position/exposure limits
  - utilization figures

#### `POST /api/v1/risk/killswitch`

- auth: required
- toggles kill switch state
- body typically includes:

```json
{
  "active": true,
  "reason": "operator action"
}
```

### Settings

#### `GET /api/v1/settings`

- auth: required
- returns:
  - LLM default provider and tier model selection
  - provider configuration state with masked API-key info
  - risk threshold settings
  - system environment/version/uptime
  - configured broker summary

#### `PUT /api/v1/settings`

- auth: required
- updates the memory-backed settings service
- important caveat: changes do not persist across server restart

### Events

#### `GET /api/v1/events`

- auth: required
- returns persisted event records

### Conversations

#### `GET /api/v1/conversations`

- auth: required
- list conversations

#### `POST /api/v1/conversations`

- auth: required
- create a conversation record

#### `GET /api/v1/conversations/{id}/messages`

- auth: required
- list messages for a conversation

#### `POST /api/v1/conversations/{id}/messages`

- auth: required
- append a message to a conversation

### Audit log

#### `GET /api/v1/audit-log`

- auth: required
- lists audit entries for operational actions and state changes

## WebSocket reference

Endpoint:

```text
GET /ws
```

### Client commands

The server accepts JSON messages shaped like:

```json
{
  "action": "subscribe",
  "strategy_ids": ["..."],
  "run_ids": ["..."]
}
```

Supported actions:

- `subscribe`
- `unsubscribe`
- `subscribe_all`
- `unsubscribe_all`

Acknowledgement format:

```json
{
  "status": "ok",
  "action": "subscribe"
}
```

Error format:

```json
{
  "type": "error",
  "error": "invalid command JSON"
}
```

### Subscription semantics

- `strategy_ids` subscribes to events for one or more strategies
- `run_ids` subscribes to a specific pipeline run
- `subscribe_all` turns on broadcast delivery for all events

### Broadcast event types

The server broadcasts event envelopes for things such as:

- pipeline start
- signal emission
- order submitted
- position update

The frontend run-detail page subscribes by `run_id` while a run is active.

## Notable API caveats

- WebSocket auth is not enforced by the current handler.
- Backtest-related code exists in the repo, but the main API server does not yet expose a supported backtest route surface.
- Settings writes are session-scoped, not durable.
