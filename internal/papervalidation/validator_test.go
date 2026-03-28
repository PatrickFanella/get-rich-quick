package papervalidation

import (
	"math"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
)

func TestDefaultThresholds(t *testing.T) {
	t.Parallel()

	th := DefaultThresholds()

	if th.MinSharpeRatio != 1.0 {
		t.Errorf("MinSharpeRatio = %f, want 1.0", th.MinSharpeRatio)
	}
	if th.MaxDrawdown != 0.15 {
		t.Errorf("MaxDrawdown = %f, want 0.15", th.MaxDrawdown)
	}
	if th.MinWinRate != 0.40 {
		t.Errorf("MinWinRate = %f, want 0.40", th.MinWinRate)
	}
	if th.MinProfitFactor != 1.5 {
		t.Errorf("MinProfitFactor = %f, want 1.5", th.MinProfitFactor)
	}
	if th.MinRoundTripTrades != 20 {
		t.Errorf("MinRoundTripTrades = %d, want 20", th.MinRoundTripTrades)
	}
	if th.MinCalendarDays != 60 {
		t.Errorf("MinCalendarDays = %d, want 60", th.MinCalendarDays)
	}
}

func TestValidateAllPassingGo(t *testing.T) {
	t.Parallel()

	metrics := backtest.Metrics{
		SharpeRatio:  1.5,
		MaxDrawdown:  0.10,
		WinRate:      0.55,
		ProfitFactor: 2.0,
	}
	analytics := backtest.TradeAnalytics{ClosedTrades: 25}
	th := DefaultThresholds()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(61 * 24 * time.Hour) // 61 days later

	result := Validate(metrics, analytics, th, start, now)

	if !result.AllPassed {
		t.Error("AllPassed = false, want true when all metrics exceed thresholds")
	}
	if !result.GoDecision {
		t.Error("GoDecision = false, want true when all metrics pass and 60 days elapsed")
	}
	if result.ElapsedDays != 61 {
		t.Errorf("ElapsedDays = %d, want 61", result.ElapsedDays)
	}
	if len(result.Metrics) != 5 {
		t.Fatalf("len(Metrics) = %d, want 5", len(result.Metrics))
	}
	for _, m := range result.Metrics {
		if !m.Passed {
			t.Errorf("metric %q passed = false, want true", m.Name)
		}
	}
}

func TestValidateNoGoInsufficientDays(t *testing.T) {
	t.Parallel()

	metrics := backtest.Metrics{
		SharpeRatio:  1.5,
		MaxDrawdown:  0.10,
		WinRate:      0.55,
		ProfitFactor: 2.0,
	}
	analytics := backtest.TradeAnalytics{ClosedTrades: 25}
	th := DefaultThresholds()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(30 * 24 * time.Hour) // only 30 days

	result := Validate(metrics, analytics, th, start, now)

	if !result.AllPassed {
		t.Error("AllPassed = false, want true (metrics pass)")
	}
	if result.GoDecision {
		t.Error("GoDecision = true, want false (insufficient days)")
	}
	if result.ElapsedDays != 30 {
		t.Errorf("ElapsedDays = %d, want 30", result.ElapsedDays)
	}
}

func TestValidateNoGoFailedMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		metrics  backtest.Metrics
		analytics backtest.TradeAnalytics
		failedMetric string
	}{
		{
			name: "low sharpe",
			metrics: backtest.Metrics{
				SharpeRatio:  0.8,
				MaxDrawdown:  0.10,
				WinRate:      0.55,
				ProfitFactor: 2.0,
			},
			analytics:    backtest.TradeAnalytics{ClosedTrades: 25},
			failedMetric: "sharpe_ratio",
		},
		{
			name: "high drawdown",
			metrics: backtest.Metrics{
				SharpeRatio:  1.5,
				MaxDrawdown:  0.20,
				WinRate:      0.55,
				ProfitFactor: 2.0,
			},
			analytics:    backtest.TradeAnalytics{ClosedTrades: 25},
			failedMetric: "max_drawdown",
		},
		{
			name: "low win rate",
			metrics: backtest.Metrics{
				SharpeRatio:  1.5,
				MaxDrawdown:  0.10,
				WinRate:      0.30,
				ProfitFactor: 2.0,
			},
			analytics:    backtest.TradeAnalytics{ClosedTrades: 25},
			failedMetric: "win_rate",
		},
		{
			name: "low profit factor",
			metrics: backtest.Metrics{
				SharpeRatio:  1.5,
				MaxDrawdown:  0.10,
				WinRate:      0.55,
				ProfitFactor: 1.2,
			},
			analytics:    backtest.TradeAnalytics{ClosedTrades: 25},
			failedMetric: "profit_factor",
		},
		{
			name: "insufficient trades",
			metrics: backtest.Metrics{
				SharpeRatio:  1.5,
				MaxDrawdown:  0.10,
				WinRate:      0.55,
				ProfitFactor: 2.0,
			},
			analytics:    backtest.TradeAnalytics{ClosedTrades: 10},
			failedMetric: "round_trip_trades",
		},
	}

	th := DefaultThresholds()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(61 * 24 * time.Hour)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := Validate(tt.metrics, tt.analytics, th, start, now)

			if result.AllPassed {
				t.Error("AllPassed = true, want false")
			}
			if result.GoDecision {
				t.Error("GoDecision = true, want false")
			}

			found := false
			for _, m := range result.Metrics {
				if m.Name == tt.failedMetric {
					found = true
					if m.Passed {
						t.Errorf("metric %q passed = true, want false", tt.failedMetric)
					}
				}
			}
			if !found {
				t.Errorf("metric %q not found in results", tt.failedMetric)
			}
		})
	}
}

func TestValidateZeroDates(t *testing.T) {
	t.Parallel()

	metrics := backtest.Metrics{
		SharpeRatio:  1.5,
		MaxDrawdown:  0.10,
		WinRate:      0.55,
		ProfitFactor: 2.0,
	}
	analytics := backtest.TradeAnalytics{ClosedTrades: 25}
	th := DefaultThresholds()

	result := Validate(metrics, analytics, th, time.Time{}, time.Time{})

	if result.ElapsedDays != 0 {
		t.Errorf("ElapsedDays = %d, want 0 for zero dates", result.ElapsedDays)
	}
	if result.GoDecision {
		t.Error("GoDecision = true, want false for zero dates")
	}
}

func TestValidateExactThresholds(t *testing.T) {
	t.Parallel()

	// Values exactly at thresholds — should NOT pass (thresholds are strict
	// inequalities: > or <, not >= or <=).
	metrics := backtest.Metrics{
		SharpeRatio:  1.0,
		MaxDrawdown:  0.15,
		WinRate:      0.40,
		ProfitFactor: 1.5,
	}
	analytics := backtest.TradeAnalytics{ClosedTrades: 20}
	th := DefaultThresholds()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(61 * 24 * time.Hour)

	result := Validate(metrics, analytics, th, start, now)

	// Sharpe == 1.0 does not satisfy > 1.0, so it fails.
	if result.AllPassed {
		t.Error("AllPassed = true, want false for exact-threshold values")
	}
	if result.GoDecision {
		t.Error("GoDecision = true, want false for exact-threshold values")
	}

	// round_trip_trades uses >=, so 20 should pass.
	for _, m := range result.Metrics {
		if m.Name == "round_trip_trades" && !m.Passed {
			t.Error("round_trip_trades should pass with exactly 20 trades (>= threshold)")
		}
	}
}

func TestValidateInfiniteProfitFactor(t *testing.T) {
	t.Parallel()

	metrics := backtest.Metrics{
		SharpeRatio:  1.5,
		MaxDrawdown:  0.10,
		WinRate:      0.55,
		ProfitFactor: math.Inf(1),
	}
	analytics := backtest.TradeAnalytics{ClosedTrades: 25}
	th := DefaultThresholds()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := start.Add(61 * 24 * time.Hour)

	result := Validate(metrics, analytics, th, start, now)

	if !result.AllPassed {
		t.Error("AllPassed = false, want true (infinite profit factor should pass)")
	}
}
