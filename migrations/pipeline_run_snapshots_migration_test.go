package migrations_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPipelineRunSnapshotsUpMigrationDefinesExpectedSchema(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000010_pipeline_run_snapshots.up.sql"))

	expectedFragments := []string{
		"create table pipeline_run_snapshots (",
		"id uuid primary key default gen_random_uuid()",
		"pipeline_run_id uuid not null",
		"data_type text not null check (data_type in ('market', 'news', 'fundamentals', 'social'))",
		"payload jsonb not null",
		"created_at timestamptz not null default now()",
		"create index idx_pipeline_run_snapshots_pipeline_run_id on pipeline_run_snapshots (pipeline_run_id)",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}
}

func TestPipelineRunSnapshotsDownMigrationDropsPipelineRunSnapshotsTable(t *testing.T) {
	downSQL := normalizeSQL(t, readMigrationFile(t, "000010_pipeline_run_snapshots.down.sql"))

	if !strings.Contains(downSQL, "drop table if exists pipeline_run_snapshots cascade;") {
		t.Fatalf("expected down migration to drop pipeline_run_snapshots table, got:\n%s", downSQL)
	}
}

func TestPipelineRunSnapshotsMigrationAppliesAgainstExistingSchema(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping migration integration test in short mode")
	}

	databaseURL := os.Getenv("DB_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("skipping migration integration test: DB_URL or DATABASE_URL is not set")
	}

	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to create admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	if _, err := adminPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("failed to ensure pgcrypto extension: %v", err)
	}

	schemaName := "migr_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	sanitizedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+sanitizedSchemaName); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := adminPool.Exec(ctx, `DROP SCHEMA IF EXISTS `+sanitizedSchemaName+` CASCADE`); err != nil {
			t.Errorf("failed to drop schema %q: %v", schemaName, err)
		}
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("failed to parse database config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("failed to create schema-scoped pool: %v", err)
	}
	t.Cleanup(pool.Close)

	for _, filename := range []string{
		"000001_initial_schema.up.sql",
		"000002_historical_ohlcv.up.sql",
		"000003_backtest_configs.up.sql",
		"000004_backtest_runs.up.sql",
		"000005_backtest_config_schedule.up.sql",
		"000006_api_keys.up.sql",
		"000007_users.up.sql",
		"000008_agent_decisions_prompt_cost.up.sql",
		"000009_conversations.up.sql",
		"000009_agent_events.up.sql",
		"000010_pipeline_run_snapshots.up.sql",
	} {
		if _, err := pool.Exec(ctx, readMigrationFile(t, filename)); err != nil {
			t.Fatalf("failed to apply %s: %v", filename, err)
		}
	}

	assertTableColumns(t, ctx, pool, "pipeline_run_snapshots", map[string]columnInfo{
		"id": {
			dataType:      "uuid",
			nullable:      "NO",
			defaultClause: "gen_random_uuid()",
		},
		"pipeline_run_id": {
			dataType: "uuid",
			nullable: "NO",
		},
		"data_type": {
			dataType: "text",
			nullable: "NO",
		},
		"payload": {
			dataType: "jsonb",
			nullable: "NO",
		},
		"created_at": {
			dataType:      "timestamp with time zone",
			nullable:      "NO",
			defaultClause: "now()",
		},
	})

	assertIndexExists(t, ctx, pool, "pipeline_run_snapshots", "idx_pipeline_run_snapshots_pipeline_run_id")

	strategyID := uuid.New()
	pipelineRunID := uuid.New()
	tradeDate := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	startedAt := tradeDate.Add(14 * time.Hour)

	if _, err := pool.Exec(ctx, `
INSERT INTO strategies (id, name, ticker, market_type)
VALUES ($1, $2, $3, $4)
`, strategyID, "Snapshot strategy", "AAPL", "stock"); err != nil {
		t.Fatalf("failed to insert strategy: %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO pipeline_runs (id, strategy_id, ticker, trade_date, started_at)
VALUES ($1, $2, $3, $4, $5)
`, pipelineRunID, strategyID, "AAPL", tradeDate, startedAt); err != nil {
		t.Fatalf("failed to insert pipeline run: %v", err)
	}

	payload := `{"headline":"Market opens higher","confidence":0.87}`
	if _, err := pool.Exec(ctx, `
INSERT INTO pipeline_run_snapshots (pipeline_run_id, data_type, payload)
VALUES ($1, $2, $3::jsonb)
`, pipelineRunID, "news", payload); err != nil {
		t.Fatalf("failed to insert pipeline snapshot: %v", err)
	}

	var storedPayload string
	if err := pool.QueryRow(ctx, `
SELECT payload::text
FROM pipeline_run_snapshots
WHERE pipeline_run_id = $1
`, pipelineRunID).Scan(&storedPayload); err != nil {
		t.Fatalf("failed to query pipeline snapshot payload: %v", err)
	}
	if !strings.Contains(storedPayload, `"headline": "Market opens higher"`) {
		t.Fatalf("expected stored payload to contain headline, got %q", storedPayload)
	}

	var dataTypeErr *pgconn.PgError
	if _, err := pool.Exec(ctx, `
INSERT INTO pipeline_run_snapshots (pipeline_run_id, data_type, payload)
VALUES ($1, $2, $3::jsonb)
`, pipelineRunID, "invalid", `{}`); err == nil {
		t.Fatal("expected invalid data_type insert to fail")
	} else if !strings.Contains(err.Error(), "pipeline_run_snapshots_data_type_check") &&
		!strings.Contains(err.Error(), "check constraint") &&
		(!errors.As(err, &dataTypeErr) || dataTypeErr.Code != "23514") {
		t.Fatalf("expected invalid data_type insert to fail with check constraint, got: %v", err)
	}

	if _, err := pool.Exec(ctx, readMigrationFile(t, "000010_pipeline_run_snapshots.down.sql")); err != nil {
		t.Fatalf("failed to apply 000010_pipeline_run_snapshots.down.sql: %v", err)
	}

	assertTableDropped(t, ctx, pool, "pipeline_run_snapshots")
}
