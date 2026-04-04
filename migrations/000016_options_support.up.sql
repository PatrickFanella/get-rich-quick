-- Add options market type.
ALTER TYPE market_type ADD VALUE IF NOT EXISTS 'options';

-- Orders: add options fields.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS asset_class TEXT NOT NULL DEFAULT 'equity';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS underlying_ticker TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS option_type TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS strike NUMERIC(20, 8);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS expiry DATE;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS contract_multiplier NUMERIC(10, 4) DEFAULT 100;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS position_intent TEXT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS leg_group_id UUID;

-- Positions: add options fields.
ALTER TABLE positions ADD COLUMN IF NOT EXISTS asset_class TEXT NOT NULL DEFAULT 'equity';
ALTER TABLE positions ADD COLUMN IF NOT EXISTS underlying_ticker TEXT;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS option_type TEXT;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS strike NUMERIC(20, 8);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS expiry DATE;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS contract_multiplier NUMERIC(10, 4) DEFAULT 100;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS leg_group_id UUID;
ALTER TABLE positions ADD COLUMN IF NOT EXISTS delta NUMERIC(10, 6);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS gamma NUMERIC(10, 6);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS theta NUMERIC(10, 6);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS vega NUMERIC(10, 6);

-- Trades: add options fields.
ALTER TABLE trades ADD COLUMN IF NOT EXISTS asset_class TEXT NOT NULL DEFAULT 'equity';
ALTER TABLE trades ADD COLUMN IF NOT EXISTS open_close TEXT;
ALTER TABLE trades ADD COLUMN IF NOT EXISTS contract_multiplier NUMERIC(10, 4) DEFAULT 100;
ALTER TABLE trades ADD COLUMN IF NOT EXISTS premium NUMERIC(20, 8);

-- Option contracts cache table.
CREATE TABLE IF NOT EXISTS option_contracts (
    occ_symbol       TEXT PRIMARY KEY,
    underlying       TEXT NOT NULL,
    option_type      TEXT NOT NULL CHECK (option_type IN ('call', 'put')),
    strike           NUMERIC(20, 8) NOT NULL,
    expiry           DATE NOT NULL,
    multiplier       NUMERIC(10, 4) NOT NULL DEFAULT 100,
    style            TEXT NOT NULL DEFAULT 'american',
    fetched_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_option_contracts_underlying ON option_contracts (underlying);
CREATE INDEX IF NOT EXISTS idx_option_contracts_expiry ON option_contracts (expiry);
CREATE INDEX IF NOT EXISTS idx_orders_leg_group ON orders (leg_group_id) WHERE leg_group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_positions_leg_group ON positions (leg_group_id) WHERE leg_group_id IS NOT NULL;
