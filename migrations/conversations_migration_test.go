package migrations_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConversationsUpMigrationDefinesExpectedSchema(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000009_conversations.up.sql"))

	expectedFragments := []string{
		"create table conversations (",
		"id uuid primary key default gen_random_uuid()",
		"pipeline_run_id uuid not null",
		"agent_role text not null",
		"title text",
		"created_at timestamptz not null default now()",
		"updated_at timestamptz not null default now()",
		"create table conversation_messages (",
		"conversation_id uuid not null references conversations (id) on delete cascade",
		"role text not null check (role in ('user', 'assistant'))",
		"content text not null",
		"create index idx_conversations_pipeline_run_id on conversations (pipeline_run_id)",
		"create index idx_conversations_created_at on conversations (created_at)",
		"create index idx_conversation_messages_conversation_id on conversation_messages (conversation_id)",
		"create index idx_conversation_messages_created_at on conversation_messages (created_at)",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}
}

func TestConversationsDownMigrationDropsConversationTables(t *testing.T) {
	downSQL := normalizeSQL(t, readMigrationFile(t, "000009_conversations.down.sql"))

	for _, fragment := range []string{
		"drop table if exists conversation_messages cascade;",
		"drop table if exists conversations cascade;",
	} {
		if !strings.Contains(downSQL, fragment) {
			t.Fatalf("expected down migration to contain %q, got:\n%s", fragment, downSQL)
		}
	}
}

func TestConversationsMigrationAppliesAgainstExistingSchema(t *testing.T) {
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
	} {
		if _, err := pool.Exec(ctx, readMigrationFile(t, filename)); err != nil {
			t.Fatalf("failed to apply %s: %v", filename, err)
		}
	}

	assertTableColumns(t, ctx, pool, "conversations", map[string]columnInfo{
		"id": {
			dataType:      "uuid",
			nullable:      "NO",
			defaultClause: "gen_random_uuid()",
		},
		"pipeline_run_id": {
			dataType: "uuid",
			nullable: "NO",
		},
		"agent_role": {
			dataType: "text",
			nullable: "NO",
		},
		"title": {
			dataType: "text",
			nullable: "YES",
		},
		"created_at": {
			dataType:      "timestamp with time zone",
			nullable:      "NO",
			defaultClause: "now()",
		},
		"updated_at": {
			dataType:      "timestamp with time zone",
			nullable:      "NO",
			defaultClause: "now()",
		},
	})

	assertTableColumns(t, ctx, pool, "conversation_messages", map[string]columnInfo{
		"id": {
			dataType:      "uuid",
			nullable:      "NO",
			defaultClause: "gen_random_uuid()",
		},
		"conversation_id": {
			dataType: "uuid",
			nullable: "NO",
		},
		"role": {
			dataType: "text",
			nullable: "NO",
		},
		"content": {
			dataType: "text",
			nullable: "NO",
		},
		"created_at": {
			dataType:      "timestamp with time zone",
			nullable:      "NO",
			defaultClause: "now()",
		},
	})

	assertIndexExists(t, ctx, pool, "conversations", "idx_conversations_pipeline_run_id")
	assertIndexExists(t, ctx, pool, "conversations", "idx_conversations_created_at")
	assertIndexExists(t, ctx, pool, "conversation_messages", "idx_conversation_messages_conversation_id")
	assertIndexExists(t, ctx, pool, "conversation_messages", "idx_conversation_messages_created_at")

	var conversationID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversations (pipeline_run_id, agent_role, title)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, uuid.New(), "researcher", "Test conversation").Scan(&conversationID); err != nil {
		t.Fatalf("failed to insert conversation: %v", err)
	}

	var messageID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO conversation_messages (conversation_id, role, content)
		VALUES ($1, $2, $3)
		RETURNING id::text
	`, conversationID, "user", "hello").Scan(&messageID); err != nil {
		t.Fatalf("failed to insert conversation message: %v", err)
	}

	var roleErr *pgconn.PgError
	if _, err := pool.Exec(ctx, `
		INSERT INTO conversation_messages (conversation_id, role, content)
		VALUES ($1, $2, $3)
	`, conversationID, "system", "not allowed"); err == nil {
		t.Fatal("expected invalid role insert to fail")
	} else if !strings.Contains(err.Error(), "conversation_messages_role_check") &&
		!strings.Contains(err.Error(), "check constraint") &&
		(!errors.As(err, &roleErr) || roleErr.Code != "23514") {
		t.Fatalf("expected invalid role insert to fail with check constraint, got: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM conversations WHERE id = $1`, conversationID); err != nil {
		t.Fatalf("failed to delete conversation: %v", err)
	}

	var messageCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM conversation_messages WHERE id = $1`, messageID).Scan(&messageCount); err != nil {
		t.Fatalf("failed to count messages after conversation delete: %v", err)
	}
	if messageCount != 0 {
		t.Fatalf("expected conversation message to be cascade-deleted, got count=%d", messageCount)
	}

	if _, err := pool.Exec(ctx, readMigrationFile(t, "000009_conversations.down.sql")); err != nil {
		t.Fatalf("failed to apply 000009_conversations.down.sql: %v", err)
	}

	assertTableDropped(t, ctx, pool, "conversation_messages")
	assertTableDropped(t, ctx, pool, "conversations")
}

type columnInfo struct {
	dataType      string
	nullable      string
	defaultClause string
}

func assertTableColumns(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string, expectedColumns map[string]columnInfo) {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT column_name, data_type, is_nullable, COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		t.Fatalf("failed to query %s columns: %v", tableName, err)
	}
	defer rows.Close()

	columns := make(map[string]columnInfo)
	for rows.Next() {
		var (
			name          string
			dataType      string
			nullable      string
			defaultClause string
		)
		if err := rows.Scan(&name, &dataType, &nullable, &defaultClause); err != nil {
			t.Fatalf("failed to scan %s column: %v", tableName, err)
		}
		columns[name] = columnInfo{
			dataType:      dataType,
			nullable:      nullable,
			defaultClause: strings.ToLower(defaultClause),
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate %s columns: %v", tableName, err)
	}

	if len(columns) != len(expectedColumns) {
		t.Fatalf("expected %d %s columns, got %d", len(expectedColumns), tableName, len(columns))
	}

	for name, expected := range expectedColumns {
		got, ok := columns[name]
		if !ok {
			t.Fatalf("expected column %q to exist in %s", name, tableName)
		}
		if got.dataType != expected.dataType {
			t.Fatalf("expected %s.%s to have data type %q, got %q", tableName, name, expected.dataType, got.dataType)
		}
		if got.nullable != expected.nullable {
			t.Fatalf("expected %s.%s nullable=%q, got %q", tableName, name, expected.nullable, got.nullable)
		}
		if expected.defaultClause != "" && !strings.Contains(got.defaultClause, expected.defaultClause) {
			t.Fatalf("expected %s.%s default to contain %q, got %q", tableName, name, expected.defaultClause, got.defaultClause)
		}
	}
}

func assertIndexExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName, indexName string) {
	t.Helper()

	var count int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND tablename = $1
		  AND indexname = $2
	`, tableName, indexName).Scan(&count); err != nil {
		t.Fatalf("failed to query %s index %s: %v", tableName, indexName, err)
	}
	if count == 0 {
		t.Fatalf("expected index %s on %s", indexName, tableName)
	}
}

func assertTableDropped(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) {
	t.Helper()

	var droppedTable *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.' || $1)::text`, tableName).Scan(&droppedTable); err != nil {
		t.Fatalf("failed to verify %s removal: %v", tableName, err)
	}
	if droppedTable != nil {
		t.Fatalf("expected %s table to be dropped, got %q", tableName, *droppedTable)
	}
}
