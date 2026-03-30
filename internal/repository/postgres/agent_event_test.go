package postgres

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

func TestBuildAgentEventListQuery_NoFilters(t *testing.T) {
	query, args := buildAgentEventListQuery(repository.AgentEventFilter{}, 10, 0)

	if len(args) != 2 {
		t.Fatalf("expected 2 args (limit, offset), got %d", len(args))
	}

	if args[0] != 10 {
		t.Errorf("expected limit=10, got %v", args[0])
	}

	if args[1] != 0 {
		t.Errorf("expected offset=0, got %v", args[1])
	}

	assertContains(t, query, "FROM agent_events")
	assertContains(t, query, "ORDER BY created_at DESC, id DESC")
	assertContains(t, query, "LIMIT $1 OFFSET $2")
	assertNotContains(t, query, "WHERE")
}

func TestBuildAgentEventListQuery_AllFilters(t *testing.T) {
	runID := uuid.New()
	strategyID := uuid.New()
	after := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)

	filter := repository.AgentEventFilter{
		PipelineRunID: &runID,
		StrategyID:    &strategyID,
		AgentRole:     domain.AgentRoleTrader,
		EventKind:     "phase.started",
		Tags:          []string{"phase", "trading"},
		CreatedAfter:  &after,
		CreatedBefore: &before,
	}

	query, args := buildAgentEventListQuery(filter, 25, 50)

	if len(args) != 9 {
		t.Fatalf("expected 9 args, got %d: %v", len(args), args)
	}

	assertContains(t, query, "pipeline_run_id = $1")
	assertContains(t, query, "strategy_id = $2")
	assertContains(t, query, "agent_role = $3")
	assertContains(t, query, "event_kind = $4")
	assertContains(t, query, "tags && $5")
	assertContains(t, query, "created_at >= $6")
	assertContains(t, query, "created_at <= $7")
	assertContains(t, query, "LIMIT $8 OFFSET $9")
}

func TestBuildAgentEventListQuery_PartialFilters(t *testing.T) {
	filter := repository.AgentEventFilter{
		EventKind: "analysis.report",
		Tags:      []string{"analysis"},
	}

	query, args := buildAgentEventListQuery(filter, 10, 5)

	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}

	assertContains(t, query, "event_kind = $1")
	assertContains(t, query, "tags && $2")
	assertNotContains(t, query, "pipeline_run_id =")
	assertNotContains(t, query, "strategy_id =")
	assertContains(t, query, "LIMIT $3 OFFSET $4")
}

func TestMarshalAgentEventMetadata_ValidJSON(t *testing.T) {
	input := json.RawMessage(`{"step":"analysis"}`)

	got, err := marshalAgentEventMetadata(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(got) != `{"step":"analysis"}` {
		t.Errorf("expected metadata pass-through, got %s", got)
	}
}

func TestMarshalAgentEventMetadata_NilDefaultsToNull(t *testing.T) {
	got, err := marshalAgentEventMetadata(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != nil {
		t.Errorf("expected nil metadata, got %s", got)
	}
}

func TestMarshalAgentEventMetadata_InvalidJSON(t *testing.T) {
	_, err := marshalAgentEventMetadata(json.RawMessage(`{not valid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestAgentEventRepoIntegration_CreatePersistsEvent(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newAgentEventIntegrationPool(t, ctx)
	defer cleanup()

	repo := NewAgentEventRepo(pool)
	runID := uuid.New()
	strategyID := uuid.New()

	event := &domain.AgentEvent{
		PipelineRunID: &runID,
		StrategyID:    &strategyID,
		AgentRole:     domain.AgentRoleTrader,
		EventKind:     "phase.started",
		Title:         "Trading phase started",
		Summary:       "Trader entered the execution phase",
		Tags:          []string{"phase", "trading"},
		Metadata:      json.RawMessage(`{"step":"trading","status":"started"}`),
	}

	if err := repo.Create(ctx, event); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if event.ID == uuid.Nil {
		t.Fatal("expected Create() to populate ID")
	}
	if event.CreatedAt.IsZero() {
		t.Fatal("expected Create() to populate CreatedAt")
	}

	got, err := repo.List(ctx, repository.AgentEventFilter{PipelineRunID: &runID}, 10, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}

	persisted := got[0]
	if persisted.ID != event.ID {
		t.Errorf("ID: want %s, got %s", event.ID, persisted.ID)
	}
	if persisted.PipelineRunID == nil || *persisted.PipelineRunID != runID {
		t.Fatalf("PipelineRunID: want %s, got %v", runID, persisted.PipelineRunID)
	}
	if persisted.StrategyID == nil || *persisted.StrategyID != strategyID {
		t.Fatalf("StrategyID: want %s, got %v", strategyID, persisted.StrategyID)
	}
	if persisted.AgentRole != event.AgentRole {
		t.Errorf("AgentRole: want %q, got %q", event.AgentRole, persisted.AgentRole)
	}
	if persisted.EventKind != event.EventKind {
		t.Errorf("EventKind: want %q, got %q", event.EventKind, persisted.EventKind)
	}
	if persisted.Title != event.Title {
		t.Errorf("Title: want %q, got %q", event.Title, persisted.Title)
	}
	if persisted.Summary != event.Summary {
		t.Errorf("Summary: want %q, got %q", event.Summary, persisted.Summary)
	}
	if !reflect.DeepEqual(persisted.Tags, event.Tags) {
		t.Errorf("Tags: want %v, got %v", event.Tags, persisted.Tags)
	}
	if !jsonBytesEqual(persisted.Metadata, event.Metadata) {
		t.Errorf("Metadata: want %s, got %s", event.Metadata, persisted.Metadata)
	}
}

func TestAgentEventRepoIntegration_ListFilters(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := newAgentEventIntegrationPool(t, ctx)
	defer cleanup()

	repo := NewAgentEventRepo(pool)

	runIDOne := uuid.New()
	runIDTwo := uuid.New()
	strategyIDOne := uuid.New()
	strategyIDTwo := uuid.New()

	eventOne := insertAgentEventRow(t, ctx, pool, agentEventRow{
		PipelineRunID: &runIDOne,
		StrategyID:    &strategyIDOne,
		AgentRole:     domain.AgentRoleTrader,
		EventKind:     "phase.started",
		Title:         "Trading started",
		Tags:          []string{"phase", "trading"},
		Metadata:      json.RawMessage(`{"step":"trading"}`),
		CreatedAt:     time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC),
	})
	eventTwo := insertAgentEventRow(t, ctx, pool, agentEventRow{
		PipelineRunID: &runIDOne,
		StrategyID:    &strategyIDOne,
		AgentRole:     domain.AgentRoleTrader,
		EventKind:     "phase.completed",
		Title:         "Trading completed",
		Tags:          []string{"phase", "complete"},
		CreatedAt:     time.Date(2026, time.January, 10, 13, 0, 0, 0, time.UTC),
	})
	eventThree := insertAgentEventRow(t, ctx, pool, agentEventRow{
		PipelineRunID: &runIDTwo,
		StrategyID:    &strategyIDOne,
		AgentRole:     domain.AgentRoleMarketAnalyst,
		EventKind:     "analysis.report",
		Title:         "Analysis published",
		Tags:          []string{"analysis", "report"},
		CreatedAt:     time.Date(2026, time.January, 11, 12, 0, 0, 0, time.UTC),
	})
	eventFour := insertAgentEventRow(t, ctx, pool, agentEventRow{
		StrategyID: &strategyIDTwo,
		EventKind:  "system.heartbeat",
		Title:      "Heartbeat",
		Tags:       []string{"system"},
		CreatedAt:  time.Date(2026, time.January, 12, 12, 0, 0, 0, time.UTC),
	})

	all, err := repo.List(ctx, repository.AgentEventFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("List(all) error = %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 events, got %d", len(all))
	}
	if all[0].ID != eventFour.ID || all[3].ID != eventOne.ID {
		t.Fatalf("expected results ordered by created_at desc, got IDs %s ... %s", all[0].ID, all[3].ID)
	}

	byRun, err := repo.List(ctx, repository.AgentEventFilter{PipelineRunID: &runIDOne}, 10, 0)
	if err != nil {
		t.Fatalf("List(PipelineRunID) error = %v", err)
	}
	if len(byRun) != 2 {
		t.Fatalf("expected 2 events for run %s, got %d", runIDOne, len(byRun))
	}

	byStrategy, err := repo.List(ctx, repository.AgentEventFilter{StrategyID: &strategyIDOne}, 10, 0)
	if err != nil {
		t.Fatalf("List(StrategyID) error = %v", err)
	}
	if len(byStrategy) != 3 {
		t.Fatalf("expected 3 events for strategy %s, got %d", strategyIDOne, len(byStrategy))
	}

	byRole, err := repo.List(ctx, repository.AgentEventFilter{AgentRole: domain.AgentRoleTrader}, 10, 0)
	if err != nil {
		t.Fatalf("List(AgentRole) error = %v", err)
	}
	if len(byRole) != 2 {
		t.Fatalf("expected 2 trader events, got %d", len(byRole))
	}

	byKind, err := repo.List(ctx, repository.AgentEventFilter{EventKind: eventThree.EventKind}, 10, 0)
	if err != nil {
		t.Fatalf("List(EventKind) error = %v", err)
	}
	if len(byKind) != 1 || byKind[0].ID != eventThree.ID {
		t.Fatalf("expected only eventThree for event_kind filter, got %+v", byKind)
	}

	byTags, err := repo.List(ctx, repository.AgentEventFilter{Tags: []string{"analysis"}}, 10, 0)
	if err != nil {
		t.Fatalf("List(Tags) error = %v", err)
	}
	if len(byTags) != 1 || byTags[0].ID != eventThree.ID {
		t.Fatalf("expected only eventThree for tag filter, got %+v", byTags)
	}

	after := time.Date(2026, time.January, 10, 12, 30, 0, 0, time.UTC)
	byAfter, err := repo.List(ctx, repository.AgentEventFilter{CreatedAfter: &after}, 10, 0)
	if err != nil {
		t.Fatalf("List(CreatedAfter) error = %v", err)
	}
	if len(byAfter) != 3 {
		t.Fatalf("expected 3 events after %s, got %d", after, len(byAfter))
	}

	before := time.Date(2026, time.January, 11, 0, 0, 0, 0, time.UTC)
	byBefore, err := repo.List(ctx, repository.AgentEventFilter{CreatedBefore: &before}, 10, 0)
	if err != nil {
		t.Fatalf("List(CreatedBefore) error = %v", err)
	}
	if len(byBefore) != 2 {
		t.Fatalf("expected 2 events before %s, got %d", before, len(byBefore))
	}

	combined, err := repo.List(ctx, repository.AgentEventFilter{
		PipelineRunID: &runIDOne,
		Tags:          []string{"complete"},
		CreatedAfter:  &after,
	}, 10, 0)
	if err != nil {
		t.Fatalf("List(combined filters) error = %v", err)
	}
	if len(combined) != 1 || combined[0].ID != eventTwo.ID {
		t.Fatalf("expected only eventTwo for combined filter, got %+v", combined)
	}
}

type agentEventRow struct {
	PipelineRunID *uuid.UUID
	StrategyID    *uuid.UUID
	AgentRole     domain.AgentRole
	EventKind     string
	Title         string
	Summary       string
	Tags          []string
	Metadata      json.RawMessage
	CreatedAt     time.Time
}

func insertAgentEventRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, row agentEventRow) domain.AgentEvent {
	t.Helper()

	var event domain.AgentEvent
	metadata, err := marshalAgentEventMetadata(row.Metadata)
	if err != nil {
		t.Fatalf("marshalAgentEventMetadata() error = %v", err)
	}

	err = pool.QueryRow(ctx,
		`INSERT INTO agent_events (
			pipeline_run_id, strategy_id, agent_role, event_kind, title, summary, tags, metadata, created_at
		)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		row.PipelineRunID,
		row.StrategyID,
		nullString(row.AgentRole.String()),
		row.EventKind,
		row.Title,
		nullString(row.Summary),
		row.Tags,
		metadata,
		row.CreatedAt,
	).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		t.Fatalf("failed to insert agent event row: %v", err)
	}

	event.PipelineRunID = row.PipelineRunID
	event.StrategyID = row.StrategyID
	event.AgentRole = row.AgentRole
	event.EventKind = row.EventKind
	event.Title = row.Title
	event.Summary = row.Summary
	event.Tags = append([]string(nil), row.Tags...)
	event.Metadata = row.Metadata

	return event
}

func newAgentEventIntegrationPool(t *testing.T, ctx context.Context) (*pgxpool.Pool, func()) {
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

	schemaName := "integration_agent_event_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA "`+schemaName+`"`); err != nil {
		adminPool.Close()
		t.Fatalf("failed to create test schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schemaName+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("failed to parse pool config: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schemaName+`" CASCADE`)
		adminPool.Close()
		t.Fatalf("failed to create test pool: %v", err)
	}

	ddl := []string{
		`CREATE TABLE agent_events (
			id              UUID        NOT NULL DEFAULT gen_random_uuid(),
			pipeline_run_id UUID,
			strategy_id     UUID,
			agent_role      TEXT,
			event_kind      TEXT        NOT NULL,
			title           TEXT        NOT NULL,
			summary         TEXT,
			tags            TEXT[],
			metadata        JSONB,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (id, created_at)
		) PARTITION BY RANGE (created_at)`,
		`CREATE TABLE agent_events_2026_q1 PARTITION OF agent_events
			FOR VALUES FROM ('2026-01-01') TO ('2026-04-01')`,
		`CREATE TABLE agent_events_2026_q2 PARTITION OF agent_events
			FOR VALUES FROM ('2026-04-01') TO ('2026-07-01')`,
		`CREATE TABLE agent_events_default PARTITION OF agent_events DEFAULT`,
		`CREATE INDEX idx_agent_events_pipeline_run_id ON agent_events (pipeline_run_id)`,
		`CREATE INDEX idx_agent_events_event_kind ON agent_events (event_kind)`,
		`CREATE INDEX idx_agent_events_created_at ON agent_events (created_at)`,
		`CREATE INDEX idx_agent_events_tags_gin ON agent_events USING GIN (tags)`,
	}

	for _, stmt := range ddl {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			pool.Close()
			_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schemaName+`" CASCADE`)
			adminPool.Close()
			t.Fatalf("failed to apply test schema DDL: %v", err)
		}
	}

	cleanup := func() {
		pool.Close()
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA "`+schemaName+`" CASCADE`)
		adminPool.Close()
	}

	return pool, cleanup
}
