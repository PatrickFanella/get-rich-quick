# ADR-007: Deployment topology and scaling strategy

- **Status:** accepted
- **Date:** 2026-03-27
- **Deciders:** Engineering
- **Technical Story:** [#110](https://github.com/PatrickFanella/get-rich-quick/issues/110)

## Context

The system currently deploys as a single Go binary alongside PostgreSQL and Redis via Docker Compose. As the number of concurrent strategies grows, we need a scaling plan that avoids premature complexity while leaving a clear upgrade path.

The Go binary already handles concurrency via goroutines (parallel analysts in errgroup, cron scheduler for multiple strategies, non-blocking event channels). Each pipeline run is ~30 seconds of LLM API latency with minimal CPU/memory usage on the Go side.

## Decision

### Phase 5 (now): Single binary, Docker Compose

Stay with the current single-binary architecture. No horizontal scaling, no Kubernetes.

**Rationale:**

- The bottleneck is LLM API latency, not compute. One Go binary can run 10+ strategies concurrently with <100MB RAM.
- Paper trading validation targets 1-5 strategies. Scaling infrastructure would be premature.
- Docker Compose provides health checks, restart policies, and volume management — sufficient for single-host deployment.

### Scaling thresholds and upgrade path

| Trigger                                  | Action                                                         |
| ---------------------------------------- | -------------------------------------------------------------- |
| >50 concurrent strategies                | Split scheduler and API into separate processes                |
| Dashboard queries slow down pipeline DB  | Add PostgreSQL read replica for dashboard/API reads            |
| LLM response cache misses expensive      | Make Redis required (currently optional) for shared LLM cache  |
| Multi-region or high availability needed | Migrate from Docker Compose to Kubernetes                      |
| Multiple scheduler instances needed      | Add leader election (PostgreSQL advisory locks or Redis-based) |

### Connection pooling

- **pgx pool size:** Default 10. Increase to 25 if running >20 strategies. One pool for the single binary; if split into scheduler + API, each gets its own pool of 10.
- **Redis:** Keep optional for Phase 5. Make required when LLM response caching matters for cost control (avoid redundant $10-40 pipeline runs during paper trading sweeps).

### Infrastructure per phase

| Component     | Phase 5                    | Phase 6+ (if needed)                            |
| ------------- | -------------------------- | ----------------------------------------------- |
| App           | Single binary              | Separate scheduler + API processes              |
| Database      | Single PostgreSQL 17       | + read replica for dashboard                    |
| Cache         | Redis (optional)           | Redis (required for LLM cache sharing)          |
| Orchestration | Docker Compose             | Docker Compose; K8s only if multi-region        |
| Scaling       | Vertical (bigger instance) | Horizontal (leader-elected scheduler instances) |

## Consequences

- No operational complexity from distributed systems during paper trading validation. The team focuses on trading logic, not infrastructure.
- The single binary means a scheduler crash also takes down the API. Acceptable during paper trading; address in Phase 6 by splitting processes if stability requirements increase.
- The upgrade path is incremental — each scaling trigger has a clear, independent action. No big-bang migration required.
