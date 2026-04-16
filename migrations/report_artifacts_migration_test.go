package migrations_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReportArtifactsUpMigrationDefinesExpectedSchema(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000029_report_artifacts.up.sql"))

	for _, fragment := range []string{
		"create table report_artifacts (",
		"id uuid primary key default gen_random_uuid()",
		"strategy_id uuid not null references strategies(id)",
		"report_type text not null default 'paper_validation'",
		"time_bucket timestamptz not null",
		"status text not null default 'pending'",
		"report_json jsonb",
		"provider text",
		"model text",
		"prompt_tokens int default 0",
		"completion_tokens int default 0",
		"latency_ms int default 0",
		"error_message text",
		"created_at timestamptz not null default now()",
		"completed_at timestamptz",
		"unique (strategy_id, report_type, time_bucket)",
		"create index idx_report_artifacts_strategy_type",
		"on report_artifacts (strategy_id, report_type, completed_at desc)",
	} {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}
}

func TestReportArtifactsDownMigrationDropsTable(t *testing.T) {
	downSQL := normalizeSQL(t, readMigrationFile(t, "000029_report_artifacts.down.sql"))

	if !strings.Contains(downSQL, "drop table if exists report_artifacts") {
		t.Fatalf("expected down migration to drop report_artifacts table, got:\n%s", downSQL)
	}
}

func TestReportArtifactsMigrationAppliesAgainstExistingSchema(t *testing.T) {
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

	for _, filename := range sortedUpMigrationsThrough(t, "000029_report_artifacts.up.sql") {
		if _, err := pool.Exec(ctx, readMigrationFile(t, filename)); err != nil {
			t.Fatalf("failed to apply %s: %v", filename, err)
		}
	}

	assertTableColumns(t, ctx, pool, "report_artifacts", map[string]columnInfo{
		"id": {
			dataType:      "uuid",
			nullable:      "NO",
			defaultClause: "gen_random_uuid()",
		},
		"strategy_id": {
			dataType: "uuid",
			nullable: "NO",
		},
		"report_type": {
			dataType:      "text",
			nullable:      "NO",
			defaultClause: "paper_validation",
		},
		"time_bucket": {
			dataType: "timestamp with time zone",
			nullable: "NO",
		},
		"status": {
			dataType:      "text",
			nullable:      "NO",
			defaultClause: "pending",
		},
		"report_json": {
			dataType: "jsonb",
			nullable: "YES",
		},
		"provider": {
			dataType: "text",
			nullable: "YES",
		},
		"model": {
			dataType: "text",
			nullable: "YES",
		},
		"prompt_tokens": {
			dataType:      "integer",
			nullable:      "YES",
			defaultClause: "0",
		},
		"completion_tokens": {
			dataType:      "integer",
			nullable:      "YES",
			defaultClause: "0",
		},
		"latency_ms": {
			dataType:      "integer",
			nullable:      "YES",
			defaultClause: "0",
		},
		"error_message": {
			dataType: "text",
			nullable: "YES",
		},
		"created_at": {
			dataType:      "timestamp with time zone",
			nullable:      "NO",
			defaultClause: "now()",
		},
		"completed_at": {
			dataType: "timestamp with time zone",
			nullable: "YES",
		},
	})

	assertIndexExists(t, ctx, pool, "report_artifacts", "idx_report_artifacts_strategy_type")

	// Verify the unique constraint: a second insert on the same
	// (strategy_id, report_type, time_bucket) should fail.
	strategyID := uuid.New()
	if _, err := pool.Exec(ctx, `
INSERT INTO strategies (id, name, ticker, market_type)
VALUES ($1, 'Report Strategy', 'AAPL', 'stock')
`, strategyID); err != nil {
		t.Fatalf("failed to seed strategy: %v", err)
	}

	timeBucket := "2026-04-16T17:00:00Z"
	if _, err := pool.Exec(ctx, `
INSERT INTO report_artifacts (strategy_id, report_type, time_bucket, status)
VALUES ($1, 'paper_validation', $2::timestamptz, 'pending')
`, strategyID, timeBucket); err != nil {
		t.Fatalf("failed to insert first artifact: %v", err)
	}

	// Second insert on same key should conflict with the unique constraint.
	_, err = pool.Exec(ctx, `
INSERT INTO report_artifacts (strategy_id, report_type, time_bucket, status)
VALUES ($1, 'paper_validation', $2::timestamptz, 'completed')
`, strategyID, timeBucket)
	if err == nil {
		t.Fatal("expected unique constraint violation on duplicate (strategy_id, report_type, time_bucket), got nil")
	}
	if !strings.Contains(err.Error(), "unique") && !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected unique/duplicate error, got: %v", err)
	}

	// Apply down migration and verify table is gone.
	if _, err := pool.Exec(ctx, readMigrationFile(t, "000029_report_artifacts.down.sql")); err != nil {
		t.Fatalf("failed to apply down migration: %v", err)
	}
	assertTableDropped(t, ctx, pool, "report_artifacts")
}
