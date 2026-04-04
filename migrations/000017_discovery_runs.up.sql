CREATE TABLE IF NOT EXISTS discovery_runs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    config        JSONB       NOT NULL,
    result        JSONB       NOT NULL,
    started_at    TIMESTAMPTZ NOT NULL,
    completed_at  TIMESTAMPTZ,
    duration_ns   BIGINT,
    candidates    INT         NOT NULL DEFAULT 0,
    deployed      INT         NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_runs_started ON discovery_runs (started_at DESC);
