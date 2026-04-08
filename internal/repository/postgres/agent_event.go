package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// AgentEventRepo implements repository.AgentEventRepository using PostgreSQL.
type AgentEventRepo struct {
	pool *pgxpool.Pool
}

// Compile-time check that AgentEventRepo satisfies AgentEventRepository.
var _ repository.AgentEventRepository = (*AgentEventRepo)(nil)

const agentEventSelectSQL = `SELECT id, pipeline_run_id, strategy_id, agent_role, event_kind, title, summary, tags, metadata, created_at FROM agent_events`

// NewAgentEventRepo returns an AgentEventRepo backed by the given connection pool.
func NewAgentEventRepo(pool *pgxpool.Pool) *AgentEventRepo {
	return &AgentEventRepo{pool: pool}
}

// Create inserts a new agent event and populates the generated ID and CreatedAt
// on the provided struct.
func (r *AgentEventRepo) Create(ctx context.Context, event *domain.AgentEvent) error {
	metadata, err := marshalAgentEventMetadata(event.Metadata)
	if err != nil {
		return err
	}

	row := r.pool.QueryRow(ctx,
		`INSERT INTO agent_events (
			pipeline_run_id, strategy_id, agent_role, event_kind, title, summary, tags, metadata
		)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at`,
		event.PipelineRunID,
		event.StrategyID,
		nullString(event.AgentRole.String()),
		event.EventKind,
		event.Title,
		nullString(event.Summary),
		event.Tags,
		metadata,
	)

	if err := row.Scan(&event.ID, &event.CreatedAt); err != nil {
		return fmt.Errorf("postgres: create agent event: %w", err)
	}

	return nil
}

// List returns agent events that match the provided filter, ordered by
// created_at descending, then id descending.
func (r *AgentEventRepo) List(ctx context.Context, filter repository.AgentEventFilter, limit, offset int) ([]domain.AgentEvent, error) {
	query, args := buildAgentEventListQuery(filter, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: list agent events: %w", err)
	}
	defer rows.Close()

	var events []domain.AgentEvent
	for rows.Next() {
		event, err := scanAgentEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan agent event: %w", err)
		}
		events = append(events, *event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate agent event rows: %w", err)
	}

	return events, nil
}

// Count returns the total number of events matching the filter (ignoring pagination).
func (r *AgentEventRepo) Count(ctx context.Context, filter repository.AgentEventFilter) (int, error) {
	query, args := buildAgentEventCountQuery(filter)
	var total int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("postgres: count agent events: %w", err)
	}
	return total, nil
}

func buildAgentEventCountQuery(filter repository.AgentEventFilter) (string, []any) {
	var (
		conditions []string
		args       []any
		argIdx     int
	)
	nextArg := func(v any) string {
		argIdx++
		args = append(args, v)
		return fmt.Sprintf("$%d", argIdx)
	}
	if filter.PipelineRunID != nil {
		conditions = append(conditions, "pipeline_run_id = "+nextArg(*filter.PipelineRunID))
	}
	if filter.StrategyID != nil {
		conditions = append(conditions, "strategy_id = "+nextArg(*filter.StrategyID))
	}
	if filter.AgentRole != "" {
		conditions = append(conditions, "agent_role = "+nextArg(filter.AgentRole))
	}
	if filter.EventKind != "" {
		conditions = append(conditions, "event_kind = "+nextArg(filter.EventKind))
	}
	if len(filter.Tags) > 0 {
		conditions = append(conditions, "tags && "+nextArg(filter.Tags))
	}
	if filter.CreatedAfter != nil {
		conditions = append(conditions, "created_at >= "+nextArg(*filter.CreatedAfter))
	}
	if filter.CreatedBefore != nil {
		conditions = append(conditions, "created_at <= "+nextArg(*filter.CreatedBefore))
	}
	query := `SELECT COUNT(*) FROM agent_events`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	return query, args
}

func buildAgentEventListQuery(filter repository.AgentEventFilter, limit, offset int) (string, []any) {
	var (
		conditions []string
		args       []any
		argIdx     int
	)

	nextArg := func(v any) string {
		argIdx++
		args = append(args, v)
		return fmt.Sprintf("$%d", argIdx)
	}

	if filter.PipelineRunID != nil {
		conditions = append(conditions, "pipeline_run_id = "+nextArg(*filter.PipelineRunID))
	}

	if filter.StrategyID != nil {
		conditions = append(conditions, "strategy_id = "+nextArg(*filter.StrategyID))
	}

	if filter.AgentRole != "" {
		conditions = append(conditions, "agent_role = "+nextArg(filter.AgentRole))
	}

	if filter.EventKind != "" {
		conditions = append(conditions, "event_kind = "+nextArg(filter.EventKind))
	}

	if len(filter.Tags) > 0 {
		conditions = append(conditions, "tags && "+nextArg(filter.Tags))
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, "created_at >= "+nextArg(*filter.CreatedAfter))
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, "created_at <= "+nextArg(*filter.CreatedBefore))
	}

	query := agentEventSelectSQL
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC, id DESC"
	query += fmt.Sprintf(" LIMIT %s OFFSET %s", nextArg(limit), nextArg(offset))

	return query, args
}

func scanAgentEvent(sc scanner) (*domain.AgentEvent, error) {
	var (
		event         domain.AgentEvent
		pipelineRunID *uuid.UUID
		strategyID    *uuid.UUID
		agentRole     *string
		summary       *string
		tags          []string
		metadata      []byte
	)

	err := sc.Scan(
		&event.ID,
		&pipelineRunID,
		&strategyID,
		&agentRole,
		&event.EventKind,
		&event.Title,
		&summary,
		&tags,
		&metadata,
		&event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if pipelineRunID != nil {
		id := *pipelineRunID
		event.PipelineRunID = &id
	}

	if strategyID != nil {
		id := *strategyID
		event.StrategyID = &id
	}

	if agentRole != nil {
		event.AgentRole = domain.AgentRole(*agentRole)
	}

	if summary != nil {
		event.Summary = *summary
	}

	if tags != nil {
		event.Tags = append([]string(nil), tags...)
	}

	if metadata != nil {
		event.Metadata = json.RawMessage(metadata)
	}

	return &event, nil
}

func marshalAgentEventMetadata(data json.RawMessage) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if !json.Valid(data) {
		return nil, fmt.Errorf("postgres: agent event metadata is not valid JSON")
	}

	return data, nil
}
