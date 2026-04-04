-- Drop indexes.
DROP INDEX IF EXISTS idx_positions_leg_group;
DROP INDEX IF EXISTS idx_orders_leg_group;
DROP INDEX IF EXISTS idx_option_contracts_expiry;
DROP INDEX IF EXISTS idx_option_contracts_underlying;

-- Drop option contracts table.
DROP TABLE IF EXISTS option_contracts;

-- Trades: remove options columns.
ALTER TABLE trades DROP COLUMN IF EXISTS premium;
ALTER TABLE trades DROP COLUMN IF EXISTS contract_multiplier;
ALTER TABLE trades DROP COLUMN IF EXISTS open_close;
ALTER TABLE trades DROP COLUMN IF EXISTS asset_class;

-- Positions: remove options columns.
ALTER TABLE positions DROP COLUMN IF EXISTS vega;
ALTER TABLE positions DROP COLUMN IF EXISTS theta;
ALTER TABLE positions DROP COLUMN IF EXISTS gamma;
ALTER TABLE positions DROP COLUMN IF EXISTS delta;
ALTER TABLE positions DROP COLUMN IF EXISTS leg_group_id;
ALTER TABLE positions DROP COLUMN IF EXISTS contract_multiplier;
ALTER TABLE positions DROP COLUMN IF EXISTS expiry;
ALTER TABLE positions DROP COLUMN IF EXISTS strike;
ALTER TABLE positions DROP COLUMN IF EXISTS option_type;
ALTER TABLE positions DROP COLUMN IF EXISTS underlying_ticker;
ALTER TABLE positions DROP COLUMN IF EXISTS asset_class;

-- Orders: remove options columns.
ALTER TABLE orders DROP COLUMN IF EXISTS leg_group_id;
ALTER TABLE orders DROP COLUMN IF EXISTS position_intent;
ALTER TABLE orders DROP COLUMN IF EXISTS contract_multiplier;
ALTER TABLE orders DROP COLUMN IF EXISTS expiry;
ALTER TABLE orders DROP COLUMN IF EXISTS strike;
ALTER TABLE orders DROP COLUMN IF EXISTS option_type;
ALTER TABLE orders DROP COLUMN IF EXISTS underlying_ticker;
ALTER TABLE orders DROP COLUMN IF EXISTS asset_class;

-- NOTE: Cannot remove enum value 'options' from market_type in PostgreSQL.
-- It will remain but be unused after rollback.
