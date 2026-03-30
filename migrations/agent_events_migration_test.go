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

func TestAgentEventsUpMigrationDefinesExpectedSchema(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000009_agent_events.up.sql"))

	expectedFragments := []string{
		"create table agent_events (",
		"id uuid not null default gen_random_uuid()",
		"pipeline_run_id uuid",
		"strategy_id uuid",
		"agent_role text",
		"event_kind text not null",
		"title text not null",
		"summary text",
		"tags text[]",
		"metadata jsonb",
		"created_at timestamptz not null default now()",
		"primary key (id, created_at)",
		") partition by range (created_at);",
		"create table agent_events_2026_q1 partition of agent_events for values from ('2026-01-01') to ('2026-04-01');",
		"create table agent_events_2026_q2 partition of agent_events for values from ('2026-04-01') to ('2026-07-01');",
		"create table agent_events_default partition of agent_events default;",
		"create index idx_agent_events_pipeline_run_id on agent_events (pipeline_run_id);",
		"create index idx_agent_events_event_kind on agent_events (event_kind);",
		"create index idx_agent_events_created_at on agent_events (created_at);",
		"create index idx_agent_events_tags_gin on agent_events using gin (tags);",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}
}

func TestAgentEventsDownMigrationDropsAgentEventsTable(t *testing.T) {
	downSQL := normalizeSQL(t, readMigrationFile(t, "000009_agent_events.down.sql"))

	if !strings.Contains(downSQL, "drop table if exists agent_events cascade;") {
		t.Fatalf("expected down migration to drop agent_events table, got:\n%s", downSQL)
	}
}

func TestAgentEventsMigrationAppliesAgainstExistingSchema(t *testing.T) {
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
		"000009_agent_events.up.sql",
	} {
		if _, err := pool.Exec(ctx, readMigrationFile(t, filename)); err != nil {
			t.Fatalf("failed to apply %s: %v", filename, err)
		}
	}

	var agentEventsTable string
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.agent_events')::text`).Scan(&agentEventsTable); err != nil {
		t.Fatalf("failed to check agent_events table: %v", err)
	}
	if agentEventsTable != "agent_events" {
		t.Fatalf("expected agent_events table to exist, got %q", agentEventsTable)
	}

	type columnInfo struct {
		dataType      string
		nullable      bool
		defaultClause string
	}

	rows, err := pool.Query(ctx, `
		SELECT
			a.attname,
			pg_catalog.format_type(a.atttypid, a.atttypmod),
			NOT a.attnotnull AS nullable,
			COALESCE(pg_get_expr(d.adbin, d.adrelid), '')
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE n.nspname = current_schema()
		  AND c.relname = 'agent_events'
		  AND a.attnum > 0
		  AND NOT a.attisdropped
		ORDER BY a.attnum
	`)
	if err != nil {
		t.Fatalf("failed to query agent_events columns: %v", err)
	}
	defer rows.Close()

	columns := make(map[string]columnInfo)
	for rows.Next() {
		var (
			name          string
			dataType      string
			nullable      bool
			defaultClause string
		)
		if err := rows.Scan(&name, &dataType, &nullable, &defaultClause); err != nil {
			t.Fatalf("failed to scan agent_events column: %v", err)
		}
		columns[name] = columnInfo{
			dataType:      dataType,
			nullable:      nullable,
			defaultClause: strings.ToLower(defaultClause),
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate agent_events columns: %v", err)
	}

	expectedColumns := map[string]columnInfo{
		"id": {
			dataType:      "uuid",
			nullable:      false,
			defaultClause: "gen_random_uuid()",
		},
		"pipeline_run_id": {
			dataType: "uuid",
			nullable: true,
		},
		"strategy_id": {
			dataType: "uuid",
			nullable: true,
		},
		"agent_role": {
			dataType: "text",
			nullable: true,
		},
		"event_kind": {
			dataType: "text",
			nullable: false,
		},
		"title": {
			dataType: "text",
			nullable: false,
		},
		"summary": {
			dataType: "text",
			nullable: true,
		},
		"tags": {
			dataType: "text[]",
			nullable: true,
		},
		"metadata": {
			dataType: "jsonb",
			nullable: true,
		},
		"created_at": {
			dataType:      "timestamp with time zone",
			nullable:      false,
			defaultClause: "now()",
		},
	}

	if len(columns) != len(expectedColumns) {
		t.Fatalf("expected %d agent_events columns, got %d", len(expectedColumns), len(columns))
	}

	for name, expected := range expectedColumns {
		got, ok := columns[name]
		if !ok {
			t.Fatalf("expected column %q to exist", name)
		}
		if got.dataType != expected.dataType {
			t.Fatalf("expected column %q to have data type %q, got %q", name, expected.dataType, got.dataType)
		}
		if got.nullable != expected.nullable {
			t.Fatalf("expected column %q nullable=%t, got %t", name, expected.nullable, got.nullable)
		}
		if expected.defaultClause != "" && !strings.Contains(got.defaultClause, expected.defaultClause) {
			t.Fatalf("expected column %q default to contain %q, got %q", name, expected.defaultClause, got.defaultClause)
		}
	}

	var partitionKey string
	if err := pool.QueryRow(ctx, `
		SELECT pg_get_partkeydef(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
		  AND c.relname = 'agent_events'
	`).Scan(&partitionKey); err != nil {
		t.Fatalf("failed to query agent_events partition key: %v", err)
	}
	if strings.ToLower(partitionKey) != "range (created_at)" {
		t.Fatalf("expected agent_events to be range partitioned by created_at, got %q", partitionKey)
	}

	rows, err = pool.Query(ctx, `
		SELECT child.relname
		FROM pg_inherits
		JOIN pg_class parent ON parent.oid = pg_inherits.inhparent
		JOIN pg_class child ON child.oid = pg_inherits.inhrelid
		JOIN pg_namespace parent_ns ON parent_ns.oid = parent.relnamespace
		JOIN pg_namespace child_ns ON child_ns.oid = child.relnamespace
		WHERE parent_ns.nspname = current_schema()
		  AND child_ns.nspname = current_schema()
		  AND parent.relname = 'agent_events'
		ORDER BY child.relname
	`)
	if err != nil {
		t.Fatalf("failed to query agent_events partitions: %v", err)
	}
	defer rows.Close()

	var partitions []string
	for rows.Next() {
		var partitionName string
		if err := rows.Scan(&partitionName); err != nil {
			t.Fatalf("failed to scan agent_events partition: %v", err)
		}
		partitions = append(partitions, partitionName)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate agent_events partitions: %v", err)
	}

	expectedPartitions := []string{
		"agent_events_2026_q1",
		"agent_events_2026_q2",
		"agent_events_default",
	}
	if len(partitions) != len(expectedPartitions) {
		t.Fatalf("expected %d agent_events partitions, got %d (%v)", len(expectedPartitions), len(partitions), partitions)
	}
	for i, expected := range expectedPartitions {
		if partitions[i] != expected {
			t.Fatalf("expected partition %d to be %q, got %q", i, expected, partitions[i])
		}
	}

	type indexInfo struct {
		name string
		def  string
	}

	rows, err = pool.Query(ctx, `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = current_schema()
		  AND tablename = 'agent_events'
	`)
	if err != nil {
		t.Fatalf("failed to query agent_events indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]string)
	for rows.Next() {
		var info indexInfo
		if err := rows.Scan(&info.name, &info.def); err != nil {
			t.Fatalf("failed to scan agent_events index: %v", err)
		}
		indexes[info.name] = strings.ToLower(info.def)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("failed to iterate agent_events indexes: %v", err)
	}

	expectedIndexes := map[string]string{
		"idx_agent_events_pipeline_run_id": "(pipeline_run_id)",
		"idx_agent_events_event_kind":      "(event_kind)",
		"idx_agent_events_created_at":      "(created_at)",
		"idx_agent_events_tags_gin":        "using gin (tags)",
	}
	for name, fragment := range expectedIndexes {
		indexDef, ok := indexes[name]
		if !ok {
			t.Fatalf("expected index %q to exist", name)
		}
		if !strings.Contains(indexDef, fragment) {
			t.Fatalf("expected index %q definition to contain %q, got %q", name, fragment, indexDef)
		}
	}

	if _, err := pool.Exec(ctx, readMigrationFile(t, "000009_agent_events.down.sql")); err != nil {
		t.Fatalf("failed to apply 000009_agent_events.down.sql: %v", err)
	}

	var droppedAgentEventsTable *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass(current_schema() || '.agent_events')::text`).Scan(&droppedAgentEventsTable); err != nil {
		t.Fatalf("failed to verify agent_events table removal: %v", err)
	}
	if droppedAgentEventsTable != nil {
		t.Fatalf("expected agent_events table to be dropped, got %q", *droppedAgentEventsTable)
	}
}
