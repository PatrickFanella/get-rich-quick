-- 000009_agent_events.up.sql
-- Structured agent event stream, partitioned quarterly by created_at.
-- pipeline_run_id intentionally has no formal FK because pipeline_runs uses a
-- composite primary key (id, trade_date) to support partitioning.

CREATE TABLE agent_events (
    id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    pipeline_run_id UUID,
    strategy_id     UUID,
    agent_role      TEXT,
    event_kind      TEXT        NOT NULL,
    title           TEXT        NOT NULL,
    summary         TEXT,
    tags            TEXT[],
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE TABLE agent_events_2026_q1 PARTITION OF agent_events
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');
CREATE TABLE agent_events_2026_q2 PARTITION OF agent_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');
CREATE TABLE agent_events_default PARTITION OF agent_events DEFAULT;

CREATE INDEX idx_agent_events_pipeline_run_id ON agent_events (pipeline_run_id);
CREATE INDEX idx_agent_events_event_kind ON agent_events (event_kind);
CREATE INDEX idx_agent_events_created_at ON agent_events (created_at);
CREATE INDEX idx_agent_events_tags_gin ON agent_events USING GIN (tags);
