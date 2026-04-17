package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportArtifact represents a persisted report row.
type ReportArtifact struct {
	ID               uuid.UUID       `json:"id"`
	StrategyID       uuid.UUID       `json:"strategy_id"`
	ReportType       string          `json:"report_type"`
	TimeBucket       time.Time       `json:"time_bucket"`
	Status           string          `json:"status"`
	ReportJSON       json.RawMessage `json:"report_json,omitempty"`
	Provider         string          `json:"provider,omitempty"`
	Model            string          `json:"model,omitempty"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	LatencyMs        int             `json:"latency_ms"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
}

// ReportArtifactFilter defines supported filters when listing report artifacts.
type ReportArtifactFilter struct {
	StrategyID *uuid.UUID
	ReportType string
	Status     string
}

// ReportArtifactRepo persists report artifacts to PostgreSQL.
type ReportArtifactRepo struct {
	pool *pgxpool.Pool
}

// NewReportArtifactRepo returns a new ReportArtifactRepo.
func NewReportArtifactRepo(pool *pgxpool.Pool) *ReportArtifactRepo {
	return &ReportArtifactRepo{pool: pool}
}

// Upsert inserts or updates a report artifact keyed on
// (strategy_id, report_type, time_bucket).
func (r *ReportArtifactRepo) Upsert(ctx context.Context, a *ReportArtifact) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO report_artifacts
			(id, strategy_id, report_type, time_bucket, status, report_json,
			 provider, model, prompt_tokens, completion_tokens, latency_ms,
			 error_message, completed_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 ON CONFLICT (strategy_id, report_type, time_bucket)
		 DO UPDATE SET
			status            = EXCLUDED.status,
			report_json       = EXCLUDED.report_json,
			provider          = EXCLUDED.provider,
			model             = EXCLUDED.model,
			prompt_tokens     = EXCLUDED.prompt_tokens,
			completion_tokens = EXCLUDED.completion_tokens,
			latency_ms        = EXCLUDED.latency_ms,
			error_message     = EXCLUDED.error_message,
			completed_at      = EXCLUDED.completed_at
		 RETURNING id, created_at`,
		a.ID, a.StrategyID, a.ReportType, a.TimeBucket,
		a.Status, a.ReportJSON,
		nullString(a.Provider), nullString(a.Model),
		a.PromptTokens, a.CompletionTokens, a.LatencyMs,
		nullString(a.ErrorMessage), a.CompletedAt,
	)
	return row.Scan(&a.ID, &a.CreatedAt)
}

// GetLatest returns the most recently completed report artifact for a
// strategy and report type. Returns repository.ErrNotFound when none exist.
func (r *ReportArtifactRepo) GetLatest(ctx context.Context, strategyID uuid.UUID, reportType string) (*ReportArtifact, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, strategy_id, report_type, time_bucket, status, report_json,
		        provider, model, prompt_tokens, completion_tokens, latency_ms,
		        error_message, created_at, completed_at
		 FROM report_artifacts
		 WHERE strategy_id = $1
		   AND report_type = $2
		   AND status = 'completed'
		 ORDER BY completed_at DESC
		 LIMIT 1`,
		strategyID, reportType,
	)
	a, err := scanReportArtifact(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("postgres: get latest report artifact: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("postgres: get latest report artifact: %w", err)
	}
	return a, nil
}

// List returns report artifacts matching the filter, newest first.
func (r *ReportArtifactRepo) List(ctx context.Context, filter ReportArtifactFilter, limit, offset int) ([]ReportArtifact, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, strategy_id, report_type, time_bucket, status, report_json,
	                 provider, model, prompt_tokens, completion_tokens, latency_ms,
	                 error_message, created_at, completed_at
	          FROM report_artifacts WHERE 1=1`
	var args []any
	argN := 0
	nextArg := func(v any) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	if filter.StrategyID != nil {
		query += fmt.Sprintf(" AND strategy_id = %s", nextArg(*filter.StrategyID))
	}
	if filter.ReportType != "" {
		query += fmt.Sprintf(" AND report_type = %s", nextArg(filter.ReportType))
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = %s", nextArg(filter.Status))
	}

	query += " ORDER BY time_bucket DESC"
	query += fmt.Sprintf(" LIMIT %s OFFSET %s", nextArg(limit), nextArg(offset))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: list report artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []ReportArtifact
	for rows.Next() {
		a, err := scanReportArtifact(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan report artifact: %w", err)
		}
		artifacts = append(artifacts, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: list report artifacts rows: %w", err)
	}
	return artifacts, nil
}

func scanReportArtifact(sc scanner) (*ReportArtifact, error) {
	var (
		a            ReportArtifact
		provider     *string
		model        *string
		errorMessage *string
		completedAt  *time.Time
		reportJSON   []byte
	)
	err := sc.Scan(
		&a.ID, &a.StrategyID, &a.ReportType, &a.TimeBucket,
		&a.Status, &reportJSON,
		&provider, &model,
		&a.PromptTokens, &a.CompletionTokens, &a.LatencyMs,
		&errorMessage, &a.CreatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if provider != nil {
		a.Provider = *provider
	}
	if model != nil {
		a.Model = *model
	}
	if errorMessage != nil {
		a.ErrorMessage = *errorMessage
	}
	if completedAt != nil {
		a.CompletedAt = completedAt
	}
	if reportJSON != nil {
		a.ReportJSON = reportJSON
	}
	return &a, nil
}
