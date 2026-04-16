package postgres

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newReportArtifactIntegrationPool creates an isolated schema with the full
// table DDL needed to exercise ReportArtifactRepo. Skips when short or no DB_URL.
func newReportArtifactIntegrationPool(t *testing.T, ctx context.Context) (*pgxpool.Pool, func()) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	connString := os.Getenv("DB_URL")
	if connString == "" {
		connString = os.Getenv("DATABASE_URL")
	}
	if connString == "" {
		t.Skip("skipping integration test: DB_URL or DATABASE_URL is not set")
	}

	adminPool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("failed to create admin pool: %v", err)
	}

	if _, err := adminPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		adminPool.Close()
		t.Fatalf("failed to ensure pgcrypto extension: %v", err)
	}

	schemaName := "integration_report_artifact_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	sanitizedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+sanitizedSchemaName); err != nil {
		adminPool.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+sanitizedSchemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("failed to parse pool config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+sanitizedSchemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("failed to create test pool: %v", err)
	}

	// Minimal DDL: strategies table + report_artifacts table (no full migration stack).
	for _, stmt := range []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto`,
		`CREATE TABLE strategies (
			id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			name        TEXT        NOT NULL,
			ticker      TEXT        NOT NULL,
			market_type TEXT        NOT NULL DEFAULT 'stock',
			is_active   BOOLEAN     NOT NULL DEFAULT TRUE
		)`,
		`CREATE TABLE report_artifacts (
			id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			strategy_id       UUID        NOT NULL REFERENCES strategies(id),
			report_type       TEXT        NOT NULL DEFAULT 'paper_validation',
			time_bucket       TIMESTAMPTZ NOT NULL,
			status            TEXT        NOT NULL DEFAULT 'pending',
			report_json       JSONB,
			provider          TEXT,
			model             TEXT,
			prompt_tokens     INT         DEFAULT 0,
			completion_tokens INT         DEFAULT 0,
			latency_ms        INT         DEFAULT 0,
			error_message     TEXT,
			created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			completed_at      TIMESTAMPTZ,
			UNIQUE (strategy_id, report_type, time_bucket)
		)`,
		`CREATE INDEX idx_report_artifacts_strategy_type
			ON report_artifacts (strategy_id, report_type, completed_at DESC)`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			pool.Close()
			_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+sanitizedSchemaName+` CASCADE`)
			adminPool.Close()
			t.Fatalf("failed to apply DDL: %v", err)
		}
	}

	cleanup := func() {
		pool.Close()
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+sanitizedSchemaName+` CASCADE`)
		adminPool.Close()
	}

	return pool, cleanup
}

func TestReportArtifactRepo_UpsertInserts(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	timeBucket := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)
	a := &ReportArtifact{
		StrategyID: strategyID,
		ReportType: "paper_validation",
		TimeBucket: timeBucket,
		Status:     "pending",
	}

	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if a.ID == uuid.Nil {
		t.Fatal("Upsert() did not populate ID")
	}
	if a.CreatedAt.IsZero() {
		t.Fatal("Upsert() did not populate CreatedAt")
	}
}

func TestReportArtifactRepo_UpsertIdempotency(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	timeBucket := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 4, 16, 17, 1, 0, 0, time.UTC)

	// First upsert — pending.
	a := &ReportArtifact{
		StrategyID: strategyID,
		ReportType: "paper_validation",
		TimeBucket: timeBucket,
		Status:     "pending",
	}
	if err := repo.Upsert(ctx, a); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	firstID := a.ID

	// Second upsert on same key — updates to completed.
	a2 := &ReportArtifact{
		StrategyID:       strategyID,
		ReportType:       "paper_validation",
		TimeBucket:       timeBucket,
		Status:           "completed",
		ReportJSON:       json.RawMessage(`{"score":0.92}`),
		Provider:         "openai",
		Model:            "gpt-5-mini",
		PromptTokens:     100,
		CompletionTokens: 200,
		LatencyMs:        450,
		CompletedAt:      &completedAt,
	}
	if err := repo.Upsert(ctx, a2); err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}

	// ID should be the same row (ON CONFLICT DO UPDATE returns existing id).
	if a2.ID != firstID {
		t.Fatalf("Upsert returned id %s, want %s (same row)", a2.ID, firstID)
	}

	// Confirm DB state reflects the update.
	var status string
	var reportJSON []byte
	if err := pool.QueryRow(ctx,
		`SELECT status, report_json FROM report_artifacts WHERE id = $1`, firstID,
	).Scan(&status, &reportJSON); err != nil {
		t.Fatalf("query after second Upsert() failed: %v", err)
	}
	if status != "completed" {
		t.Fatalf("status = %q, want completed", status)
	}
	if !jsonBytesEqual(reportJSON, a2.ReportJSON) {
		t.Fatalf("report_json = %s, want %s", reportJSON, a2.ReportJSON)
	}
}

func TestReportArtifactRepo_GetLatestReturnsCompleted(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	now := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)

	// Seed: one completed, one pending.
	completedAt1 := now.Add(1 * time.Minute)
	for _, a := range []*ReportArtifact{
		{
			StrategyID:  strategyID,
			TimeBucket:  now.Add(-24 * time.Hour),
			Status:      "completed",
			Provider:    "openai",
			Model:       "gpt-5-mini",
			CompletedAt: &completedAt1,
		},
		{
			StrategyID: strategyID,
			TimeBucket: now,
			Status:     "pending",
		},
	} {
		a.ReportType = "paper_validation"
		if err := repo.Upsert(ctx, a); err != nil {
			t.Fatalf("seed Upsert() error = %v", err)
		}
	}

	got, err := repo.GetLatest(ctx, strategyID, "paper_validation")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetLatest() = nil, want completed artifact")
	}
	if got.Status != "completed" {
		t.Fatalf("GetLatest().Status = %q, want completed", got.Status)
	}
	if got.Provider != "openai" {
		t.Fatalf("GetLatest().Provider = %q, want openai", got.Provider)
	}
}

func TestReportArtifactRepo_GetLatestNilWhenNone(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	got, err := repo.GetLatest(ctx, strategyID, "paper_validation")
	if err != nil {
		t.Fatalf("GetLatest() error = %v, want nil", err)
	}
	if got != nil {
		t.Fatalf("GetLatest() = %+v, want nil when no completed artifact", got)
	}
}

func TestReportArtifactRepo_GetLatestIgnoresCompletedWithNullCompletedAt(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	timeBucket := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
INSERT INTO report_artifacts
  (strategy_id, report_type, time_bucket, status, completed_at)
VALUES
  ($1, 'paper_validation', $2, 'completed', NULL)
`, strategyID, timeBucket); err != nil {
		t.Fatalf("failed to seed row: %v", err)
	}

	got, err := repo.GetLatest(ctx, strategyID, "paper_validation")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if got != nil {
		t.Fatalf("GetLatest() = %+v, want nil when completed_at is NULL", got)
	}
}

func TestReportArtifactRepo_List(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	otherStrategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	now := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)

	toSeed := []*ReportArtifact{
		{StrategyID: strategyID, TimeBucket: now, Status: "completed"},
		{StrategyID: strategyID, TimeBucket: now.Add(1 * time.Hour), Status: "pending"},
		{StrategyID: otherStrategyID, TimeBucket: now, Status: "completed"},
	}
	for i, a := range toSeed {
		a.ReportType = "paper_validation"
		if err := repo.Upsert(ctx, a); err != nil {
			t.Fatalf("seed Upsert[%d] error = %v", i, err)
		}
	}

	// Filter by strategy.
	artifacts, err := repo.List(ctx, ReportArtifactFilter{StrategyID: &strategyID}, 50, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("List() returned %d artifacts, want 2", len(artifacts))
	}

	// Filter by status.
	completed, err := repo.List(ctx, ReportArtifactFilter{Status: "completed"}, 50, 0)
	if err != nil {
		t.Fatalf("List(status=completed) error = %v", err)
	}
	if len(completed) != 2 {
		t.Fatalf("List(status=completed) returned %d, want 2", len(completed))
	}

	// Count.
	count, err := repo.Count(ctx, ReportArtifactFilter{StrategyID: &strategyID})
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("Count() = %d, want 2", count)
	}
}

func TestReportArtifactRepo_UpsertRequiresStrategyID(t *testing.T) {
	t.Parallel()

	repo := NewReportArtifactRepo(nil) // pool unused; validation happens before query
	err := repo.Upsert(context.Background(), &ReportArtifact{
		TimeBucket: time.Now(),
		Status:     "pending",
	})
	if err == nil {
		t.Fatal("Upsert() error = nil, want strategy_id required error")
	}
	if !strings.Contains(err.Error(), "strategy_id is required") {
		t.Fatalf("error %q missing 'strategy_id is required'", err.Error())
	}
}

func TestReportArtifactRepo_UpsertRejectsInvalidReportJSON(t *testing.T) {
	t.Parallel()

	repo := NewReportArtifactRepo(nil) // pool unused; validation happens before query
	err := repo.Upsert(context.Background(), &ReportArtifact{
		StrategyID: uuid.New(),
		TimeBucket: time.Now(),
		Status:     "pending",
		ReportJSON: json.RawMessage(`{"score":`),
	})
	if err == nil {
		t.Fatal("Upsert() error = nil, want invalid JSON validation error")
	}
	if !strings.Contains(err.Error(), "report_json must be valid JSON") {
		t.Fatalf("error %q missing invalid report_json validation message", err.Error())
	}
}

func TestReportArtifactRepo_GetLatestWithNullNumericFields(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newReportArtifactIntegrationPool(t, ctx)
	defer cleanup()

	strategyID := seedReportArtifactStrategy(t, ctx, pool)
	repo := NewReportArtifactRepo(pool)

	completedAt := time.Date(2026, 4, 16, 17, 1, 0, 0, time.UTC)
	timeBucket := time.Date(2026, 4, 16, 17, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
INSERT INTO report_artifacts
  (strategy_id, report_type, time_bucket, status, prompt_tokens, completion_tokens, latency_ms, completed_at)
VALUES
  ($1, 'paper_validation', $2, 'completed', NULL, NULL, NULL, $3)
`, strategyID, timeBucket, completedAt); err != nil {
		t.Fatalf("failed to seed report_artifacts row with null numeric fields: %v", err)
	}

	got, err := repo.GetLatest(ctx, strategyID, "paper_validation")
	if err != nil {
		t.Fatalf("GetLatest() error = %v", err)
	}
	if got == nil {
		t.Fatal("GetLatest() = nil, want artifact")
	}
	if got.PromptTokens != 0 || got.CompletionTokens != 0 || got.LatencyMs != 0 {
		t.Fatalf("expected null numeric fields to map to zero values, got prompt=%d completion=%d latency=%d", got.PromptTokens, got.CompletionTokens, got.LatencyMs)
	}
}

func seedReportArtifactStrategy(t *testing.T, ctx context.Context, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if _, err := pool.Exec(ctx,
		`INSERT INTO strategies (id, name, ticker, market_type) VALUES ($1, $2, $3, $4)`,
		id, "Test Strategy "+id.String(), "AAPL", "stock",
	); err != nil {
		t.Fatalf("seedReportArtifactStrategy error = %v", err)
	}
	return id
}
