package automation

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// ---------------------------------------------------------------------------
// Stubs for report job tests
// ---------------------------------------------------------------------------

type stubStrategyRepoForReports struct {
	strategies []domain.Strategy
	err        error
}

func (s *stubStrategyRepoForReports) Create(_ context.Context, _ *domain.Strategy) error {
	return nil
}
func (s *stubStrategyRepoForReports) Get(_ context.Context, id uuid.UUID) (*domain.Strategy, error) {
	for i := range s.strategies {
		if s.strategies[i].ID == id {
			return &s.strategies[i], nil
		}
	}
	return nil, repository.ErrNotFound
}
func (s *stubStrategyRepoForReports) List(_ context.Context, _ repository.StrategyFilter, _, _ int) ([]domain.Strategy, error) {
	return s.strategies, s.err
}
func (s *stubStrategyRepoForReports) Count(_ context.Context, _ repository.StrategyFilter) (int, error) {
	return len(s.strategies), nil
}
func (s *stubStrategyRepoForReports) Update(_ context.Context, _ *domain.Strategy) error { return nil }
func (s *stubStrategyRepoForReports) Delete(_ context.Context, _ uuid.UUID) error        { return nil }
func (s *stubStrategyRepoForReports) UpdateThesis(_ context.Context, _ uuid.UUID, _ json.RawMessage) error {
	return nil
}
func (s *stubStrategyRepoForReports) GetThesisRaw(_ context.Context, _ uuid.UUID) (json.RawMessage, error) {
	return nil, nil
}

type stubBacktestConfigRepo struct {
	configs []domain.BacktestConfig
}

func (s *stubBacktestConfigRepo) Create(_ context.Context, _ *domain.BacktestConfig) error {
	return nil
}
func (s *stubBacktestConfigRepo) Get(_ context.Context, _ uuid.UUID) (*domain.BacktestConfig, error) {
	if len(s.configs) > 0 {
		return &s.configs[0], nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubBacktestConfigRepo) List(_ context.Context, _ repository.BacktestConfigFilter, _, _ int) ([]domain.BacktestConfig, error) {
	return s.configs, nil
}
func (s *stubBacktestConfigRepo) Count(_ context.Context, _ repository.BacktestConfigFilter) (int, error) {
	return len(s.configs), nil
}
func (s *stubBacktestConfigRepo) Update(_ context.Context, _ *domain.BacktestConfig) error {
	return nil
}
func (s *stubBacktestConfigRepo) Delete(_ context.Context, _ uuid.UUID) error { return nil }

type stubBacktestRunRepo struct {
	runs []domain.BacktestRun
}

func (s *stubBacktestRunRepo) Create(_ context.Context, _ *domain.BacktestRun) error { return nil }
func (s *stubBacktestRunRepo) Get(_ context.Context, _ uuid.UUID) (*domain.BacktestRun, error) {
	if len(s.runs) > 0 {
		return &s.runs[0], nil
	}
	return nil, repository.ErrNotFound
}
func (s *stubBacktestRunRepo) List(_ context.Context, _ repository.BacktestRunFilter, _, _ int) ([]domain.BacktestRun, error) {
	return s.runs, nil
}
func (s *stubBacktestRunRepo) Count(_ context.Context, _ repository.BacktestRunFilter) (int, error) {
	return len(s.runs), nil
}

// stubReportArtifactRepo captures upserted artifacts in-memory.
type stubReportArtifactRepo struct {
	artifacts []reportArtifactCapture
}

type reportArtifactCapture struct {
	StrategyID uuid.UUID
	Status     string
	ReportJSON json.RawMessage
	Error      string
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPaperValidationReport_NoPaperStrategies(t *testing.T) {
	t.Parallel()

	orch := NewJobOrchestrator(OrchestratorDeps{
		StrategyRepo: &stubStrategyRepoForReports{
			strategies: []domain.Strategy{
				{ID: uuid.New(), Name: "live", Status: "active", IsPaper: false},
			},
		},
	})
	orch.registerReportJobs()

	// Job should succeed with no paper strategies — nothing to do.
	err := orch.paperValidationReport(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPaperValidationReport_NoReportArtifactRepo(t *testing.T) {
	t.Parallel()

	orch := NewJobOrchestrator(OrchestratorDeps{})
	orch.registerReportJobs()

	// registerReportJobs should be a no-op when repo is nil.
	if _, ok := orch.jobs["paper_validation_report"]; ok {
		t.Fatal("expected paper_validation_report job to NOT be registered when repo is nil")
	}
}

func TestGenerateOneReport_NoBacktestConfigs(t *testing.T) {
	t.Parallel()

	stratID := uuid.New()
	orch := newReportTestOrchestrator(
		[]domain.Strategy{{ID: stratID, Name: "test", Status: "active", IsPaper: true, CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}},
		nil, // no configs
		nil, // no runs
	)

	err := orch.generateOneReport(context.Background(), stratID, "test", time.Now().Truncate(24*time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error when no backtest configs exist")
	}
}

func TestGenerateOneReport_GenerationSucceedsButPersistFailsWithoutRepo(t *testing.T) {
	t.Parallel()

	stratID := uuid.New()
	configID := uuid.New()
	metricsJSON := mustMarshal(t, backtest.Metrics{
		TotalReturn:     0.15,
		SharpeRatio:     1.5,
		MaxDrawdown:     0.08,
		WinRate:         0.55,
		StartTime:       time.Now().Add(-30 * 24 * time.Hour),
		EndTime:         time.Now(),
		StartEquity:     10000,
		EndEquity:       11500,
		TotalBars:       30,
		Volatility:      0.20,
		ProfitFactor:    2.0,
		AvgWinLossRatio: 1.5,
		CalmarRatio:     1.8,
		SortinoRatio:    1.2,
	})
	// Empty trade log so ComputeTradeAnalytics is skipped (no +Inf).
	tradeLogJSON := json.RawMessage(`[]`)

	orch := newReportTestOrchestrator(
		[]domain.Strategy{{ID: stratID, Name: "test", Status: "active", IsPaper: true, CreatedAt: time.Now().Add(-90 * 24 * time.Hour)}},
		[]domain.BacktestConfig{{ID: configID, StrategyID: stratID}},
		[]domain.BacktestRun{{ID: uuid.New(), BacktestConfigID: configID, Metrics: metricsJSON, TradeLog: tradeLogJSON}},
	)

	// ReportArtifactRepo is nil → will fail at persist, but NOT at report generation.
	err := orch.generateOneReport(context.Background(), stratID, "test", time.Now().Truncate(24*time.Hour), time.Now())
	if err == nil {
		t.Fatal("expected error (nil repo), got nil")
	}
	// Verify it's a persist error, not a generation error.
	if contains := "persist report"; !strings.Contains(err.Error(), contains) {
		t.Fatalf("error %q should contain %q", err.Error(), contains)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newReportTestOrchestrator creates an orchestrator with stubs for report testing.
// A nil-pool ReportArtifactRepo is wired so the upsert hits real code — which
// will fail on DB call but we verify the flow up to that point via error checks.
// For the success test we need a fake; since we can't easily fake the concrete
// pgrepo type, we set it to nil and accept that generateOneReport will fail at
// persist-time. The test structure verifies the generation logic itself.
func newReportTestOrchestrator(
	strategies []domain.Strategy,
	configs []domain.BacktestConfig,
	runs []domain.BacktestRun,
) *JobOrchestrator {
	orch := NewJobOrchestrator(OrchestratorDeps{
		StrategyRepo:       &stubStrategyRepoForReports{strategies: strategies},
		BacktestConfigRepo: &stubBacktestConfigRepo{configs: configs},
		BacktestRunRepo:    &stubBacktestRunRepo{runs: runs},
		// ReportArtifactRepo is nil — generateOneReport tests use
		// persistErrorArtifact which handles nil gracefully (logs + returns origErr).
	})
	return orch
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
