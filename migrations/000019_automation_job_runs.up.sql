CREATE TABLE IF NOT EXISTS automation_job_runs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_name     TEXT NOT NULL,
    status       TEXT NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    duration_ns  BIGINT,
    result       JSONB,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_job_runs_name_started ON automation_job_runs (job_name, started_at DESC);
