package risk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// polymarketMaxExposurePct is the stricter per-market limit for Polymarket.
const polymarketMaxExposurePct = 0.05

// DefaultPositionLimits returns the default position limits.
func DefaultPositionLimits() PositionLimits {
	return PositionLimits{
		MaxPerPositionPct: 0.20,
		MaxTotalPct:       1.00,
		MaxConcurrent:     10,
		MaxPerMarketPct:   0.50,
	}
}

// engineState holds the mutable risk engine state protected by a mutex.
type engineState struct {
	mu sync.RWMutex
	cb CircuitBreakerStatus
	ks KillSwitchStatus
}

// RiskEngineImpl is the concrete implementation of RiskEngine.
type RiskEngineImpl struct {
	limits       PositionLimits
	positionRepo repository.PositionRepository
	logger       *slog.Logger
	state        engineState
}

// NewRiskEngine creates a new RiskEngineImpl.
func NewRiskEngine(limits PositionLimits, positionRepo repository.PositionRepository, logger *slog.Logger) *RiskEngineImpl {
	if logger == nil {
		logger = slog.Default()
	}
	return &RiskEngineImpl{
		limits:       limits,
		positionRepo: positionRepo,
		logger:       logger,
		state: engineState{
			cb: CircuitBreakerStatus{State: CircuitBreakerPhaseOpen},
			ks: KillSwitchStatus{Active: false},
		},
	}
}

// CheckPreTrade evaluates whether an order should be allowed before submission.
func (e *RiskEngineImpl) CheckPreTrade(ctx context.Context, order *domain.Order, portfolio Portfolio) (bool, string, error) {
	e.state.mu.RLock()
	defer e.state.mu.RUnlock()

	if e.state.ks.Active {
		return false, fmt.Sprintf("kill switch is active: %s", e.state.ks.Reason), nil
	}

	if e.state.cb.State == CircuitBreakerPhaseTripped {
		return false, fmt.Sprintf("circuit breaker tripped: %s", e.state.cb.Reason), nil
	}

	if order == nil {
		return false, "order is nil", nil
	}
	if order.Ticker == "" {
		return false, "order ticker is required", nil
	}
	if order.Quantity <= 0 {
		return false, "order quantity must be positive", nil
	}

	e.logger.InfoContext(ctx, "pre-trade check passed",
		slog.String("ticker", order.Ticker),
		slog.Float64("quantity", order.Quantity),
	)
	return true, "", nil
}

// CheckPositionLimits evaluates whether adding quantity for ticker stays within limits.
// The quantity parameter represents the additional position exposure as a fraction of
// the portfolio (e.g. 0.10 = 10%).
func (e *RiskEngineImpl) CheckPositionLimits(ctx context.Context, ticker string, quantity float64, portfolio Portfolio) (bool, string, error) {
	e.state.mu.RLock()
	defer e.state.mu.RUnlock()

	// Check max per-position size.
	currentExposure := portfolio.PositionExposureBySymbol[ticker]
	if currentExposure+quantity > e.limits.MaxPerPositionPct {
		return false, fmt.Sprintf(
			"position size %.2f%% for %s exceeds max %.2f%%",
			(currentExposure+quantity)*100, ticker, e.limits.MaxPerPositionPct*100,
		), nil
	}

	// Check max total exposure.
	if portfolio.TotalExposurePct+quantity > e.limits.MaxTotalPct {
		return false, fmt.Sprintf(
			"total exposure %.2f%% exceeds max %.2f%%",
			(portfolio.TotalExposurePct+quantity)*100, e.limits.MaxTotalPct*100,
		), nil
	}

	// Check max concurrent positions (only if opening a new position).
	if _, exists := portfolio.PositionExposureBySymbol[ticker]; !exists {
		if portfolio.ConcurrentPositions >= e.limits.MaxConcurrent {
			return false, fmt.Sprintf(
				"concurrent positions %d reached max %d",
				portfolio.ConcurrentPositions, e.limits.MaxConcurrent,
			), nil
		}
	}

	// Check per-market exposure limits.
	for market, exposure := range portfolio.MarketExposurePct {
		limit := e.limits.MaxPerMarketPct
		if market == domain.MarketTypePolymarket {
			limit = polymarketMaxExposurePct
		}
		if exposure > limit {
			return false, fmt.Sprintf(
				"%s market exposure %.2f%% exceeds max %.2f%%",
				market, exposure*100, limit*100,
			), nil
		}
	}

	e.logger.InfoContext(ctx, "position limits check passed",
		slog.String("ticker", ticker),
		slog.Float64("quantity", quantity),
	)
	return true, "", nil
}

// GetStatus returns the current engine state.
func (e *RiskEngineImpl) GetStatus(_ context.Context) (EngineStatus, error) {
	e.state.mu.RLock()
	defer e.state.mu.RUnlock()

	status := domain.RiskStatusNormal
	if e.state.cb.State == CircuitBreakerPhaseTripped {
		status = domain.RiskStatusBreached
	} else if e.state.ks.Active {
		status = domain.RiskStatusWarning
	}

	return EngineStatus{
		RiskStatus:     status,
		CircuitBreaker: e.state.cb,
		KillSwitch:     e.state.ks,
		PositionLimits: e.limits,
		UpdatedAt:      time.Now(),
	}, nil
}

// TripCircuitBreaker activates the circuit breaker.
func (e *RiskEngineImpl) TripCircuitBreaker(ctx context.Context, reason string) error {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	now := time.Now()
	e.state.cb = CircuitBreakerStatus{
		State:     CircuitBreakerPhaseTripped,
		Reason:    reason,
		TrippedAt: &now,
	}
	e.logger.WarnContext(ctx, "circuit breaker tripped", slog.String("reason", reason))
	return nil
}

// ResetCircuitBreaker resets the circuit breaker to open state.
func (e *RiskEngineImpl) ResetCircuitBreaker(ctx context.Context) error {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	e.state.cb = CircuitBreakerStatus{State: CircuitBreakerPhaseOpen}
	e.logger.InfoContext(ctx, "circuit breaker reset")
	return nil
}

// IsKillSwitchActive returns whether the kill switch is active.
func (e *RiskEngineImpl) IsKillSwitchActive(_ context.Context) (bool, error) {
	e.state.mu.RLock()
	defer e.state.mu.RUnlock()
	return e.state.ks.Active, nil
}

// ActivateKillSwitch activates the kill switch.
func (e *RiskEngineImpl) ActivateKillSwitch(ctx context.Context, reason string) error {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	now := time.Now()
	e.state.ks = KillSwitchStatus{
		Active:      true,
		Reason:      reason,
		ActivatedAt: &now,
	}
	e.logger.WarnContext(ctx, "kill switch activated", slog.String("reason", reason))
	return nil
}

// DeactivateKillSwitch deactivates the kill switch.
func (e *RiskEngineImpl) DeactivateKillSwitch(ctx context.Context) error {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()

	e.state.ks = KillSwitchStatus{Active: false}
	e.logger.InfoContext(ctx, "kill switch deactivated")
	return nil
}
