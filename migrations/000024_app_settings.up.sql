-- Single-row table for persisted application settings.
-- API keys are never stored here; only model selections, provider preferences,
-- and risk thresholds that operators adjust at runtime.
CREATE TABLE IF NOT EXISTS app_settings (
    id         SMALLINT PRIMARY KEY DEFAULT 1,
    llm_config JSONB NOT NULL DEFAULT '{}',
    risk_config JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);

-- Seed the row so there is always exactly one settings record.
INSERT INTO app_settings (id) VALUES (1) ON CONFLICT DO NOTHING;
