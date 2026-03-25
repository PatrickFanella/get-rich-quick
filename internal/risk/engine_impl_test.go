package risk

import (
	"context"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

func newTestEngine() *RiskEngineImpl {
	return NewRiskEngine(DefaultPositionLimits(), nil, nil)
}

func TestCheckPreTrade_Approved(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	order := &domain.Order{
		Ticker:   "AAPL",
		Quantity: 10,
		Side:     domain.OrderSideBuy,
	}
	portfolio := Portfolio{
		TotalExposurePct:    0.50,
		ConcurrentPositions: 5,
	}

	approved, reason, err := engine.CheckPreTrade(context.Background(), order, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatalf("expected approved, got rejected: %s", reason)
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}

func TestCheckPreTrade_KillSwitchActive(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	if err := engine.ActivateKillSwitch(context.Background(), "manual halt"); err != nil {
		t.Fatalf("unexpected error activating kill switch: %v", err)
	}

	order := &domain.Order{Ticker: "AAPL", Quantity: 10}
	approved, reason, err := engine.CheckPreTrade(context.Background(), order, Portfolio{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected when kill switch active")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPreTrade_CircuitBreakerTripped(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	if err := engine.TripCircuitBreaker(context.Background(), "loss limit"); err != nil {
		t.Fatalf("unexpected error tripping circuit breaker: %v", err)
	}

	order := &domain.Order{Ticker: "AAPL", Quantity: 10}
	approved, reason, err := engine.CheckPreTrade(context.Background(), order, Portfolio{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected when circuit breaker tripped")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPreTrade_InvalidOrder(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()

	// Nil order.
	approved, reason, err := engine.CheckPreTrade(context.Background(), nil, Portfolio{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for nil order")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason for nil order")
	}

	// Empty ticker.
	approved, reason, err = engine.CheckPreTrade(context.Background(), &domain.Order{Quantity: 10}, Portfolio{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for empty ticker")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason for empty ticker")
	}

	// Zero quantity.
	approved, reason, err = engine.CheckPreTrade(context.Background(), &domain.Order{Ticker: "AAPL", Quantity: 0}, Portfolio{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for zero quantity")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason for zero quantity")
	}
}

func TestCheckPositionLimits_WithinLimits(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.50,
		ConcurrentPositions: 3,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.10,
			"GOOG": 0.10,
			"MSFT": 0.10,
		},
		MarketExposurePct: map[domain.MarketType]float64{
			domain.MarketTypeStock: 0.30,
		},
	}

	approved, reason, err := engine.CheckPositionLimits(context.Background(), "AAPL", 0.05, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatalf("expected approved, got rejected: %s", reason)
	}
}

func TestCheckPositionLimits_ExceedsPositionSize(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.50,
		ConcurrentPositions: 3,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.15,
		},
	}

	// Adding 0.10 to existing 0.15 = 0.25, exceeds MaxPerPositionPct of 0.20.
	approved, reason, err := engine.CheckPositionLimits(context.Background(), "AAPL", 0.10, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for exceeding position size")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPositionLimits_ExceedsTotalExposure(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.95,
		ConcurrentPositions: 3,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.10,
		},
	}

	// Adding 0.10 to total 0.95 = 1.05, exceeds MaxTotalPct of 1.00.
	approved, reason, err := engine.CheckPositionLimits(context.Background(), "AAPL", 0.10, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for exceeding total exposure")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPositionLimits_ExceedsConcurrentPositions(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.50,
		ConcurrentPositions: 10,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.05, "GOOG": 0.05, "MSFT": 0.05, "AMZN": 0.05, "META": 0.05,
			"TSLA": 0.05, "NVDA": 0.05, "AMD": 0.05, "INTC": 0.05, "ORCL": 0.05,
		},
	}

	// New ticker when already at max concurrent positions.
	approved, reason, err := engine.CheckPositionLimits(context.Background(), "IBM", 0.05, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for exceeding concurrent positions")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPositionLimits_ExceedsMarketExposure(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.60,
		ConcurrentPositions: 3,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.10,
		},
		MarketExposurePct: map[domain.MarketType]float64{
			domain.MarketTypeStock: 0.55, // Exceeds MaxPerMarketPct of 0.50.
		},
	}

	approved, reason, err := engine.CheckPositionLimits(context.Background(), "AAPL", 0.05, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for exceeding market exposure")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPositionLimits_ExceedsPolymarketExposure(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.10,
		ConcurrentPositions: 1,
		PositionExposureBySymbol: map[string]float64{
			"POLY-ELECTION": 0.04,
		},
		MarketExposurePct: map[domain.MarketType]float64{
			domain.MarketTypePolymarket: 0.06, // Exceeds polymarket limit of 0.05.
		},
	}

	approved, reason, err := engine.CheckPositionLimits(context.Background(), "POLY-ELECTION", 0.01, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Fatal("expected rejected for exceeding Polymarket exposure")
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestCheckPositionLimits_ExistingPositionBypassesConcurrentCheck(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	portfolio := Portfolio{
		TotalExposurePct:    0.50,
		ConcurrentPositions: 10,
		PositionExposureBySymbol: map[string]float64{
			"AAPL": 0.05, "GOOG": 0.05, "MSFT": 0.05, "AMZN": 0.05, "META": 0.05,
			"TSLA": 0.05, "NVDA": 0.05, "AMD": 0.05, "INTC": 0.05, "ORCL": 0.05,
		},
	}

	// Adding to an existing position should not be blocked by concurrent limit.
	approved, reason, err := engine.CheckPositionLimits(context.Background(), "AAPL", 0.05, portfolio)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Fatalf("expected approved for existing position, got rejected: %s", reason)
	}
}

func TestGetStatus_Normal(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	status, err := engine.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RiskStatus != domain.RiskStatusNormal {
		t.Fatalf("expected normal status, got %q", status.RiskStatus)
	}
	if status.CircuitBreaker.State != CircuitBreakerPhaseOpen {
		t.Fatalf("expected open circuit breaker, got %q", status.CircuitBreaker.State)
	}
	if status.KillSwitch.Active {
		t.Fatal("expected kill switch inactive")
	}
	if status.PositionLimits.MaxConcurrent != 10 {
		t.Fatalf("expected max concurrent 10, got %d", status.PositionLimits.MaxConcurrent)
	}
}

func TestGetStatus_Breached(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	if err := engine.TripCircuitBreaker(context.Background(), "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, err := engine.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RiskStatus != domain.RiskStatusBreached {
		t.Fatalf("expected breached status, got %q", status.RiskStatus)
	}
}

func TestGetStatus_Warning(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	if err := engine.ActivateKillSwitch(context.Background(), "manual"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	status, err := engine.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.RiskStatus != domain.RiskStatusWarning {
		t.Fatalf("expected warning status, got %q", status.RiskStatus)
	}
}

func TestCircuitBreakerLifecycle(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	ctx := context.Background()

	// Trip.
	if err := engine.TripCircuitBreaker(ctx, "loss limit"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status, _ := engine.GetStatus(ctx)
	if status.CircuitBreaker.State != CircuitBreakerPhaseTripped {
		t.Fatalf("expected tripped, got %q", status.CircuitBreaker.State)
	}

	// Reset.
	if err := engine.ResetCircuitBreaker(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	status, _ = engine.GetStatus(ctx)
	if status.CircuitBreaker.State != CircuitBreakerPhaseOpen {
		t.Fatalf("expected open after reset, got %q", status.CircuitBreaker.State)
	}
}

func TestKillSwitchLifecycle(t *testing.T) {
	t.Parallel()

	engine := newTestEngine()
	ctx := context.Background()

	// Activate.
	if err := engine.ActivateKillSwitch(ctx, "emergency"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	active, err := engine.IsKillSwitchActive(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !active {
		t.Fatal("expected kill switch active")
	}

	// Deactivate.
	if err := engine.DeactivateKillSwitch(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	active, err = engine.IsKillSwitchActive(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if active {
		t.Fatal("expected kill switch inactive")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ RiskEngine = (*RiskEngineImpl)(nil)
}

func TestDefaultPositionLimits(t *testing.T) {
	t.Parallel()

	limits := DefaultPositionLimits()
	if limits.MaxPerPositionPct != 0.20 {
		t.Fatalf("expected 0.20, got %f", limits.MaxPerPositionPct)
	}
	if limits.MaxTotalPct != 1.00 {
		t.Fatalf("expected 1.00, got %f", limits.MaxTotalPct)
	}
	if limits.MaxConcurrent != 10 {
		t.Fatalf("expected 10, got %d", limits.MaxConcurrent)
	}
	if limits.MaxPerMarketPct != 0.50 {
		t.Fatalf("expected 0.50, got %f", limits.MaxPerMarketPct)
	}
}
