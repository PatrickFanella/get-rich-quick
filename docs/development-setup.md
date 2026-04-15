---
title: "Development Setup"
description: "Complete local development workflow for backend, frontend, database, testing, and smoke-mode execution."
status: "canonical"
updated: "2026-04-03"
tags: [development, setup, local-dev]
---

# Development Setup

This guide is for contributors who need the full day-to-day workflow rather than the shortest first-run path.

## Toolchain

Required:

- Go 1.25
- Node.js 20+
- npm
- Docker and Docker Compose v2+
- PostgreSQL client tools if you want to inspect the database outside Compose

Recommended:

- [Task](https://taskfile.dev) for the project command runner
- `jq` for API and login scripting
- `golangci-lint`

## Repository layout

These are the directories you will touch most often:

| Path | Purpose |
| --- | --- |
| `cmd/tradingagent` | app bootstrap, runtime wiring, strategy runner, docs tests |
| `internal/api` | REST API, middleware, auth, settings, WebSocket hub |
| `internal/agent` | agent runtime, config resolution, prompts, runner orchestration |
| `internal/data` | provider chains, caching, historical downloads |
| `internal/execution` | brokers, paper trading, order management |
| `internal/risk` | hard risk engine, kill switch, exposure limits |
| `internal/repository/postgres` | persistence layer |
| `web/` | React/Vite frontend |
| `migrations/` | SQL migrations |
| `docs/` | canonical docs plus archive material |

## Configuration model

The server loads configuration from environment variables through `internal/config`.

Important behavior:

- `.env` is auto-loaded only when `APP_ENV=development`.
- `JWT_SECRET` is required for the API server to start.
- most provider integrations are opt-in by key presence
- non-secret settings edited through the API/UI persist to the `app_settings` table when the DB-backed persister is wired; secrets are not written back to `.env` or stored in the database
- startup fails fast on database schema mismatch before the rest of the runtime boots; fix by running migrations, then restarting the process

Start from:

```bash
cp .env.example .env
```

Then set the minimum viable local config:

```dotenv
APP_ENV=development
JWT_SECRET=replace-this-with-a-real-secret
OPENAI_API_KEY=...
```

## Running the stack with Docker Compose

The default contributor path is:

```bash
docker compose up --build
```

That Compose stack is backend-only in current local and production wiring. Run the frontend separately from `web/`.

Or with Task:

```bash
task dev
```

Useful Compose/Task commands:

```bash
task dev
task dev:down
task dev:logs
task dev:restart
task dev:psql
```

## Running the backend natively

If you want the API server outside Docker:

1. Start PostgreSQL and Redis yourself, or run only those services via Compose.
2. Set `DATABASE_URL`, `REDIS_URL`, and `JWT_SECRET`.
3. Run migrations.
4. Start the server:

```bash
go run ./cmd/tradingagent serve
```

Or build first:

```bash
task build
./bin/tradingagent serve
```

## Running the frontend

```bash
cd web
npm install
npm run dev
```

The frontend default API base URL is `http://localhost:8080`.

The frontend is a separate Vite app. Backend root `/` is not the SPA in the current Compose or production stack.

## Database migrations

The project uses SQL migrations under `migrations/`.

Run them explicitly before expecting a new build to boot cleanly against an updated database. If the server already started and failed with a schema mismatch, apply migrations and then restart it; the mismatch is fail-fast and does not self-heal inside the running process.

Common commands:

```bash
task migrate:up
task migrate:down
task migrate:status
task migrate:create -- add_feature_name
```

The schema includes persistence for:

- strategies
- pipeline runs and phase timings
- pipeline run snapshots
- agent decisions and events
- conversations and messages
- orders, positions, trades
- memories
- market-data cache and historical OHLCV
- audit log
- users
- API keys
- backtest configs and backtest runs

## Creating a local user

There is no self-service registration flow yet. For local dev:

```bash
docker compose exec postgres psql -U postgres -d tradingagent <<'SQL'
INSERT INTO users (username, password_hash)
VALUES ('demo', crypt('demo-pass', gen_salt('bf')))
ON CONFLICT (username) DO NOTHING;
SQL
```

## Smoke mode for deterministic runs

`APP_ENV=smoke` activates a deterministic manual-run path that is useful for end-to-end testing without depending on real LLMs and live upstream providers.

Because `.env` auto-loading only happens in `development`, export your env file before starting smoke mode:

```bash
set -a
source .env
set +a
export APP_ENV=smoke
./bin/tradingagent serve
```

Smoke mode is especially useful when you want to verify:

- strategy creation
- login/auth
- manual run dispatch
- run detail pages
- event plumbing
- persistence wiring

## Testing and quality checks

Primary Task targets:

```bash
task build
task test
task test:race
task test:integration
task test:cover
task lint
task fmt
task fmt:check
task vet
task vulncheck
task audit
task check
task ci
```

Notes:

- integration tests require PostgreSQL
- the current repository contains unresolved merge conflict markers in multiple Go and TypeScript files, so some broad test/build commands may fail before your specific change is even exercised
- docs-only validation currently relies mostly on file/link checks and the dedicated docs tests in `cmd/tradingagent`

## CLI workflow

The CLI talks to the local API server. Typical env setup:

```bash
export TRADINGAGENT_API_URL=http://127.0.0.1:8080
export TRADINGAGENT_TOKEN=...
```

Examples:

```bash
./bin/tradingagent strategies list
./bin/tradingagent run AAPL
./bin/tradingagent portfolio
./bin/tradingagent risk status
./bin/tradingagent dashboard
./bin/tradingagent memories search earnings
```

Full command reference: [Reference: CLI](reference/cli.md)

## Frontend workflow

The web app lives in `web/` and exposes these routes:

- `/login`
- `/`
- `/strategies`
- `/strategies/:id`
- `/runs`
- `/runs/:id`
- `/portfolio`
- `/memories`
- `/settings`
- `/risk`
- `/realtime`

Web UI reference: [Reference: Web UI](reference/web-ui.md)

## Operational development notes

Useful health endpoints:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/health
curl http://localhost:8080/metrics
```

Useful database access:

```bash
docker compose exec postgres psql -U postgres -d tradingagent
```

Useful log inspection:

```bash
docker compose logs -f app
docker compose logs -f postgres
docker compose logs -f redis
```

## Current contributor hazards

Before doing anything expensive, read [Known Issues](known-issues.md).

The big ones today:

- unresolved merge conflicts exist in several runtime, risk, API-test, and frontend files
- some documented integrations are partially wired rather than fully productionized
- WebSocket auth is not enforced by the current handler
- secret values entered through the settings UI do not persist across restarts; non-secret settings persist through `app_settings`

## Suggested contributor reading order

1. [Getting Started](getting-started.md)
2. [Reference: Architecture](reference/architecture.md)
3. [Reference: API](reference/api.md)
4. [Reference: Agents and Runtime](reference/agents.md)
5. [Reference: Strategy Config](reference/strategy-config.md)
6. [Known Issues](known-issues.md)
