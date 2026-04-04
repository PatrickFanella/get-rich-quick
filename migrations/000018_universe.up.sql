CREATE TABLE IF NOT EXISTS universe_tickers (
    ticker       TEXT PRIMARY KEY,
    name         TEXT,
    exchange     TEXT,
    index_group  TEXT NOT NULL DEFAULT 'other',
    watch_score  NUMERIC(10, 4) NOT NULL DEFAULT 0,
    last_scanned TIMESTAMPTZ,
    active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_universe_watch_score ON universe_tickers (watch_score DESC) WHERE active;
CREATE INDEX IF NOT EXISTS idx_universe_index_group ON universe_tickers (index_group) WHERE active;
