package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReportArtifact represents a single LLM-generated report persisted to the
// report_artifacts table. The idempotency key is (strategy_id, report_type,
// time_bucket); upserts on the same key update in place.
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

// ReportArtifactRepo persists LLM-generated report artifacts to PostgreSQL.
type ReportArtifactRepo struct {
	pool *pgxpool.Pool
}

// NewReportArtifactRepo returns a new ReportArtifactRepo.
func NewReportArtifactRepo(pool *pgxpool.Pool) *ReportArtifactRepo {
	return &ReportArtifactRepo{pool: pool}
}

// Upsert inserts or updates a report artifact using the idempotency key
// (strategy_id, report_type, time_bucket). On conflict all mutable fields
// are overwritten. ID and CreatedAt are populated on insert.
func (r *ReportArtifactRepo) Upsert(ctx context.Context, a *ReportArtifact) error {
	if a.StrategyID == uuid.Nil {
		return fmt.Errorf("postgres: report artifact strategy_id is required")
	}
	if a.ReportType == "" {
		a.ReportType = "paper_validation"
	}
	if a.Status == "" {
		a.Status = "pending"
	}

	var reportJSON []byte
	if len(a.ReportJSON) > 0 {
		if !json.Valid(a.ReportJSON) {
			return fmt.Errorf("postgres: report artifact report_json must be valid JSON")
		}
		reportJSON = a.ReportJSON
	}

	row := r.pool.QueryRow(ctx, `
INSERT INTO report_artifacts
    (strategy_id, report_type, time_bucket, status, report_json,
     provider, model, prompt_tokens, completion_tokens, latency_ms,
     error_message, completed_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (strategy_id, report_type, time_bucket) DO UPDATE SET
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
		a.StrategyID, a.ReportType, a.TimeBucket, a.Status, reportJSON,
		nullString(a.Provider), nullString(a.Model),
		a.PromptTokens, a.CompletionTokens, a.LatencyMs,
		nullString(a.ErrorMessage), a.CompletedAt,
	)

	return row.Scan(&a.ID, &a.CreatedAt)
}

// GetLatest returns the most recently completed report artifact for the given
// strategy and report type. Returns (nil, nil) when no completed artifact exists.
func (r *ReportArtifactRepo) GetLatest(ctx context.Context, strategyID uuid.UUID, reportType string) (*ReportArtifact, error) {
	if reportType == "" {
		reportType = "paper_validation"
	}

	row := r.pool.QueryRow(ctx, `
SELECT id, strategy_id, report_type, time_bucket, status, report_json,
       provider, model, prompt_tokens, completion_tokens, latency_ms,
       error_message, created_at, completed_at
FROM report_artifacts
WHERE strategy_id = $1
  AND report_type = $2
  AND status = 'completed'
  AND completed_at IS NOT NULL
ORDER BY completed_at DESC NULLS LAST, created_at DESC, id DESC
LIMIT 1`,
		strategyID, reportType,
	)

	a, err := scanReportArtifact(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
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

	query := `
SELECT id, strategy_id, report_type, time_bucket, status, report_json,
       provider, model, prompt_tokens, completion_tokens, latency_ms,
       error_message, created_at, completed_at
FROM report_artifacts
WHERE 1=1`

	args := []any{}
	argN := 1

	if filter.StrategyID != nil {
		query += fmt.Sprintf(" AND strategy_id = $%d", argN)
		args = append(args, *filter.StrategyID)
		argN++
	}
	if filter.ReportType != "" {
		query += fmt.Sprintf(" AND report_type = $%d", argN)
		args = append(args, filter.ReportType)
		argN++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, filter.Status)
		argN++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

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
	return artifacts, rows.Err()
}

// Count returns the number of report artifacts matching the filter.
func (r *ReportArtifactRepo) Count(ctx context.Context, filter ReportArtifactFilter) (int, error) {
	query := `SELECT COUNT(*) FROM report_artifacts WHERE 1=1`
	args := []any{}
	argN := 1

	if filter.StrategyID != nil {
		query += fmt.Sprintf(" AND strategy_id = $%d", argN)
		args = append(args, *filter.StrategyID)
		argN++
	}
	if filter.ReportType != "" {
		query += fmt.Sprintf(" AND report_type = $%d", argN)
		args = append(args, filter.ReportType)
		argN++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, filter.Status)
		argN++
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("postgres: count report artifacts: %w", err)
	}
	return count, nil
}

func scanReportArtifact(s scanner) (*ReportArtifact, error) {
	var (
		a                ReportArtifact
		reportJSON       []byte
		provider         *string
		model            *string
		errMsg           *string
		promptTokens     *int
		completionTokens *int
		latencyMs        *int
		completedAt      *time.Time
	)

	if err := s.Scan(
		&a.ID, &a.StrategyID, &a.ReportType, &a.TimeBucket, &a.Status,
		&reportJSON, &provider, &model,
		&promptTokens, &completionTokens, &latencyMs,
		&errMsg, &a.CreatedAt, &completedAt,
	); err != nil {
		return nil, err
	}

	if len(reportJSON) > 0 {
		a.ReportJSON = json.RawMessage(reportJSON)
	}
	if provider != nil {
		a.Provider = *provider
	}
	if model != nil {
		a.Model = *model
	}
	if errMsg != nil {
		a.ErrorMessage = *errMsg
	}
	if promptTokens != nil {
		a.PromptTokens = *promptTokens
	}
	if completionTokens != nil {
		a.CompletionTokens = *completionTokens
	}
	if latencyMs != nil {
		a.LatencyMs = *latencyMs
	}
	a.CompletedAt = completedAt

	return &a, nil
}
