package scheduler

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
	"github.com/PatrickFanella/get-rich-quick/internal/risk"
)

type mockStrategyRepo struct {
	mu         sync.Mutex
	strategies []domain.Strategy
	filters    []repository.StrategyFilter
	listErr    error
}

func (m *mockStrategyRepo) Create(context.Context, *domain.Strategy) error { return nil }

func (m *mockStrategyRepo) Get(context.Context, uuid.UUID) (*domain.Strategy, error) { return nil, nil }

func (m *mockStrategyRepo) List(_ context.Context, filter repository.StrategyFilter, limit, offset int) ([]domain.Strategy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.filters = append(m.filters, filter)
	if m.listErr != nil {
		return nil, m.listErr
	}
	if offset >= len(m.strategies) {
		return nil, nil
	}

	end := offset + limit
	if end > len(m.strategies) {
		end = len(m.strategies)
	}

	return append([]domain.Strategy(nil), m.strategies[offset:end]...), nil
}

func (m *mockStrategyRepo) Update(context.Context, *domain.Strategy) error { return nil }

func (m *mockStrategyRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (m *mockStrategyRepo) lastFilter() (repository.StrategyFilter, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.filters) == 0 {
		return repository.StrategyFilter{}, false
	}

	return m.filters[len(m.filters)-1], true
}

type pipelineCall struct {
	strategyID uuid.UUID
	ticker     string
}

type mockPipeline struct {
	mu      sync.Mutex
	calls   []pipelineCall
	callsCh chan pipelineCall
	err     error
}

func (m *mockPipeline) Execute(_ context.Context, strategyID uuid.UUID, ticker string) (*agent.PipelineState, error) {
	call := pipelineCall{strategyID: strategyID, ticker: ticker}

	m.mu.Lock()
	m.calls = append(m.calls, call)
	m.mu.Unlock()

	if m.callsCh != nil {
		select {
		case m.callsCh <- call:
		default:
		}
	}

	return &agent.PipelineState{}, m.err
}

func (m *mockPipeline) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

type mockRiskEngine struct {
	killSwitchActive bool
	killSwitchErr    error
}

func (m *mockRiskEngine) CheckPreTrade(context.Context, *domain.Order, risk.Portfolio) (bool, string, error) {
	return true, "", nil
}

func (m *mockRiskEngine) CheckPositionLimits(context.Context, string, float64, risk.Portfolio) (bool, string, error) {
	return true, "", nil
}

func (m *mockRiskEngine) GetStatus(context.Context) (risk.EngineStatus, error) {
	return risk.EngineStatus{}, nil
}

func (m *mockRiskEngine) TripCircuitBreaker(context.Context, string) error { return nil }

func (m *mockRiskEngine) ResetCircuitBreaker(context.Context) error { return nil }

func (m *mockRiskEngine) IsKillSwitchActive(context.Context) (bool, error) {
	return m.killSwitchActive, m.killSwitchErr
}

func (m *mockRiskEngine) ActivateKillSwitch(context.Context, string) error { return nil }

func (m *mockRiskEngine) DeactivateKillSwitch(context.Context) error { return nil }

func (m *mockRiskEngine) UpdateMetrics(context.Context, float64, float64, int) error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSchedulerStartTriggersPipelineExecution(t *testing.T) {
	strategyID := uuid.New()
	repo := &mockStrategyRepo{
		strategies: []domain.Strategy{
			{
				ID:           strategyID,
				Ticker:       "BTCUSD",
				MarketType:   domain.MarketTypeCrypto,
				ScheduleCron: "@every 1s",
				IsActive:     true,
			},
		},
	}
	pipeline := &mockPipeline{callsCh: make(chan pipelineCall, 1)}
	s := NewScheduler(repo, pipeline, &mockRiskEngine{}, testLogger())

	if err := s.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer s.Stop()

	filter, ok := repo.lastFilter()
	if !ok {
		t.Fatal("expected strategy repository List to be called")
	}
	if filter.IsActive == nil || !*filter.IsActive {
		t.Fatalf("expected active strategy filter, got %+v", filter)
	}

	select {
	case call := <-pipeline.callsCh:
		if call.strategyID != strategyID {
			t.Fatalf("pipeline strategyID = %s, want %s", call.strategyID, strategyID)
		}
		if call.ticker != "BTCUSD" {
			t.Fatalf("pipeline ticker = %q, want %q", call.ticker, "BTCUSD")
		}
	case <-time.After(2500 * time.Millisecond):
		t.Fatal("timed out waiting for scheduled pipeline execution")
	}
}

func TestSchedulerRunStrategySkipsWhenKillSwitchActive(t *testing.T) {
	pipeline := &mockPipeline{}
	s := NewScheduler(nil, pipeline, &mockRiskEngine{killSwitchActive: true}, testLogger())

	s.runStrategy(domain.Strategy{
		ID:         uuid.New(),
		Ticker:     "BTCUSD",
		MarketType: domain.MarketTypeCrypto,
	})

	if got := pipeline.callCount(); got != 0 {
		t.Fatalf("pipeline calls = %d, want 0", got)
	}
}

func TestSchedulerRunStrategySkipsOutsideStockMarketHours(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation(America/New_York): %v", err)
	}

	pipeline := &mockPipeline{}
	s := NewScheduler(nil, pipeline, &mockRiskEngine{}, testLogger())
	s.nowFunc = func() time.Time {
		return time.Date(2024, time.January, 6, 10, 0, 0, 0, loc)
	}

	s.runStrategy(domain.Strategy{
		ID:         uuid.New(),
		Ticker:     "AAPL",
		MarketType: domain.MarketTypeStock,
	})

	if got := pipeline.callCount(); got != 0 {
		t.Fatalf("pipeline calls = %d, want 0", got)
	}
}
