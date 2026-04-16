CREATE TABLE report_artifacts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id       UUID NOT NULL REFERENCES strategies(id),
    report_type       TEXT NOT NULL DEFAULT 'paper_validation',
    time_bucket       TIMESTAMPTZ NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending',
    report_json       JSONB,
    provider          TEXT,
    model             TEXT,
    prompt_tokens     INT DEFAULT 0,
    completion_tokens INT DEFAULT 0,
    latency_ms        INT DEFAULT 0,
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at      TIMESTAMPTZ,
    UNIQUE (strategy_id, report_type, time_bucket)
);
CREATE INDEX idx_report_artifacts_strategy_type
    ON report_artifacts (strategy_id, report_type, completed_at DESC);
