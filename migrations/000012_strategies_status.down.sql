DO $$
BEGIN
    IF to_regclass(current_schema() || '.strategies') IS NOT NULL THEN
        EXECUTE 'DROP TRIGGER IF EXISTS trg_strategies_sync_status_with_is_active ON strategies;';
    END IF;
END
$$;

DROP FUNCTION IF EXISTS sync_strategy_status_with_is_active();

ALTER TABLE strategies
    DROP COLUMN IF EXISTS skip_next_run,
    DROP COLUMN IF EXISTS status;

COMMENT ON COLUMN strategies.is_active IS NULL;
