package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/papervalidation"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
	pgrepo "github.com/PatrickFanella/get-rich-quick/internal/repository/postgres"
	"github.com/PatrickFanella/get-rich-quick/internal/scheduler"
)

const reportTypePaperValidation = "paper_validation"

var paperValidationReportSpec = scheduler.ScheduleSpec{
	Type:         scheduler.ScheduleTypeAfterHours,
	Cron:         "0 17 * * 1-5", // 5 PM ET daily, after market close
	SkipWeekends: true,
	SkipHolidays: true,
}

func (o *JobOrchestrator) registerReportJobs() {
	if o.deps.ReportArtifactRepo == nil {
		o.logger.Info("report_jobs: skipped — report artifact repo not configured")
		return
	}
	o.Register(
		"paper_validation_report",
		"Generate paper-trading validation reports for active paper strategies",
		paperValidationReportSpec,
		o.paperValidationReport,
	)
}

// paperValidationReport generates a paper-validation report for every active
// paper strategy. Each strategy is processed independently — a failure in one
// does not block the others.
func (o *JobOrchestrator) paperValidationReport(ctx context.Context) error {
	o.logger.Info("paper_validation_report: starting")

	strategies, err := o.deps.StrategyRepo.List(ctx, repository.StrategyFilter{Status: "active"}, 0, 0)
	if err != nil {
		return fmt.Errorf("paper_validation_report: list strategies: %w", err)
	}

	// Filter to paper-only strategies.
	type paperEntry struct {
		ID   uuid.UUID
		Name string
	}
	var paperStrategies []paperEntry
	for _, s := range strategies {
		if s.IsPaper {
			paperStrategies = append(paperStrategies, paperEntry{ID: s.ID, Name: s.Name})
		}
	}
	if len(paperStrategies) == 0 {
		o.logger.Info("paper_validation_report: no active paper strategies")
		return nil
	}
	o.logger.Info("paper_validation_report: processing",
		slog.Int("strategies", len(paperStrategies)),
	)

	now := time.Now().UTC()
	timeBucket := now.Truncate(24 * time.Hour)
	var succeeded, failed int

	for _, ps := range paperStrategies {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Jitter: 0–119s between strategies to spread LLM/DB load.
		jitter := time.Duration(rand.IntN(120)) * time.Second
		select {
		case <-time.After(jitter):
		case <-ctx.Done():
			return ctx.Err()
		}

		if err := o.generateOneReport(ctx, ps.ID, ps.Name, timeBucket, now); err != nil {
			failed++
			if o.reportMetrics != nil {
				o.reportMetrics.RecordReportWorkerError(ps.ID.String())
			}
			o.logger.Warn("paper_validation_report: strategy failed",
				slog.String("strategy", ps.Name),
				slog.Any("error", err),
			)
		} else {
			succeeded++
			if o.reportMetrics != nil {
				o.reportMetrics.RecordReportWorkerSuccess(ps.ID.String())
			}
		}
	}

	o.logger.Info("paper_validation_report: completed",
		slog.Int("succeeded", succeeded),
		slog.Int("failed", failed),
	)
	return nil
}

// generateOneReport loads the latest backtest metrics for a single strategy,
// generates the paper-validation report, and persists the artifact.
func (o *JobOrchestrator) generateOneReport(
	ctx context.Context,
	strategyID uuid.UUID,
	strategyName string,
	timeBucket, now time.Time,
) error {
	start := time.Now()

	// Load the strategy to get paper start date.
	strategy, err := o.deps.StrategyRepo.Get(ctx, strategyID)
	if err != nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("get strategy: %w", err))
	}
	paperStart := strategy.CreatedAt

	// Find the most recent backtest config for this strategy.
	if o.deps.BacktestConfigRepo == nil || o.deps.BacktestRunRepo == nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("backtest repos not configured"))
	}

	configs, err := o.deps.BacktestConfigRepo.List(ctx, repository.BacktestConfigFilter{
		StrategyID: &strategyID,
	}, 1, 0)
	if err != nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("list backtest configs: %w", err))
	}
	if len(configs) == 0 {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("no backtest configs found for strategy %s", strategyName))
	}

	configID := configs[0].ID
	runs, err := o.deps.BacktestRunRepo.List(ctx, repository.BacktestRunFilter{
		BacktestConfigID: &configID,
	}, 1, 0)
	if err != nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("list backtest runs: %w", err))
	}
	if len(runs) == 0 {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("no backtest runs found for config %s", configID))
	}

	latestRun := runs[0]

	// Deserialise metrics.
	var btMetrics backtest.Metrics
	if err := json.Unmarshal(latestRun.Metrics, &btMetrics); err != nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("unmarshal metrics: %w", err))
	}

	// Deserialise trade analytics from trade log if available.
	var analytics backtest.TradeAnalytics
	if len(latestRun.TradeLog) > 0 {
		var trades []domain.Trade
		if err := json.Unmarshal(latestRun.TradeLog, &trades); err != nil {
			// Non-fatal: proceed with zero analytics.
			o.logger.Warn("paper_validation_report: unmarshal trade log failed, using zero analytics",
				slog.String("strategy", strategyName),
				slog.Any("error", err),
			)
		} else {
			analytics = backtest.ComputeTradeAnalytics(trades, btMetrics.StartTime, btMetrics.EndTime)
		}
	}

	// Generate the report (pure function — no LLM call).
	thresholds := papervalidation.DefaultThresholds()
	report := papervalidation.GenerateReport(btMetrics, analytics, thresholds, paperStart, now)

	reportJSON, err := json.Marshal(report)
	if err != nil {
		return o.persistErrorArtifact(ctx, strategyID, timeBucket, fmt.Errorf("marshal report: %w", err))
	}

	latencyMs := int(time.Since(start).Milliseconds())
	completed := time.Now().UTC()

	artifact := &pgrepo.ReportArtifact{
		StrategyID:  strategyID,
		ReportType:  reportTypePaperValidation,
		TimeBucket:  timeBucket,
		Status:      "completed",
		ReportJSON:  reportJSON,
		LatencyMs:   latencyMs,
		CompletedAt: &completed,
	}
	if o.deps.ReportArtifactRepo == nil {
		return fmt.Errorf("persist report: report artifact repo not configured")
	}
	if err := o.deps.ReportArtifactRepo.Upsert(ctx, artifact); err != nil {
		return fmt.Errorf("persist report: %w", err)
	}

	o.logger.Info("paper_validation_report: generated",
		slog.String("strategy", strategyName),
		slog.String("decision", report.Decision),
		slog.Int("latency_ms", latencyMs),
	)
	return nil
}

// persistErrorArtifact records a failed report attempt so the failure is
// visible in the report_artifacts table.
func (o *JobOrchestrator) persistErrorArtifact(
	ctx context.Context,
	strategyID uuid.UUID,
	timeBucket time.Time,
	origErr error,
) error {
	if o.deps.ReportArtifactRepo == nil {
		o.logger.Error("paper_validation_report: cannot persist error artifact (repo nil)",
			slog.Any("original_error", origErr),
		)
		return origErr
	}
	completed := time.Now().UTC()
	artifact := &pgrepo.ReportArtifact{
		StrategyID:   strategyID,
		ReportType:   reportTypePaperValidation,
		TimeBucket:   timeBucket,
		Status:       "error",
		ErrorMessage: origErr.Error(),
		CompletedAt:  &completed,
	}
	if err := o.deps.ReportArtifactRepo.Upsert(ctx, artifact); err != nil {
		o.logger.Error("paper_validation_report: failed to persist error artifact",
			slog.Any("original_error", origErr),
			slog.Any("persist_error", err),
		)
	}
	return origErr
}
