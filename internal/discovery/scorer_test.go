package discovery

import (
	"math"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
)

func TestScoreMetricsGoodMetrics(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()
	m := backtest.Metrics{
		TotalBars:    50,
		SharpeRatio:  1.5,
		SortinoRatio: 2.0,
		MaxDrawdown:  0.10,
	}
	score := ScoreMetrics(m, cfg)
	if math.IsInf(score, -1) {
		t.Fatal("expected positive score, got -Inf")
	}
	// 0.5*1.5 + 0.3*2.0 - 0.2*0.10 = 0.75 + 0.6 - 0.02 = 1.33
	want := 1.33
	if math.Abs(score-want) > 0.001 {
		t.Fatalf("score = %f, want ~%f", score, want)
	}
}

func TestScoreMetricsLowSharpe(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()
	m := backtest.Metrics{
		TotalBars:    50,
		SharpeRatio:  0.3, // below MinSharpe 0.5
		SortinoRatio: 1.0,
		MaxDrawdown:  0.10,
	}
	score := ScoreMetrics(m, cfg)
	if !math.IsInf(score, -1) {
		t.Fatalf("expected -Inf for low Sharpe, got %f", score)
	}
}

func TestScoreMetricsHighDrawdown(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()
	m := backtest.Metrics{
		TotalBars:    50,
		SharpeRatio:  1.5,
		SortinoRatio: 2.0,
		MaxDrawdown:  0.30, // above MaxDrawdown 0.20
	}
	score := ScoreMetrics(m, cfg)
	if !math.IsInf(score, -1) {
		t.Fatalf("expected -Inf for high drawdown, got %f", score)
	}
}

func TestScoreMetricsFewTrades(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()
	m := backtest.Metrics{
		TotalBars:    5, // below MinTrades 10
		SharpeRatio:  1.5,
		SortinoRatio: 2.0,
		MaxDrawdown:  0.10,
	}
	score := ScoreMetrics(m, cfg)
	if !math.IsInf(score, -1) {
		t.Fatalf("expected -Inf for few trades, got %f", score)
	}
}

func TestScoreMetricsNaN(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()
	m := backtest.Metrics{
		TotalBars:    50,
		SharpeRatio:  math.NaN(),
		SortinoRatio: 2.0,
		MaxDrawdown:  0.10,
	}
	score := ScoreMetrics(m, cfg)
	if !math.IsInf(score, -1) {
		t.Fatalf("expected -Inf for NaN Sharpe, got %f", score)
	}
}

func TestFilterAndRankFiltersAndSorts(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()

	results := []SweepResult{
		{Label: "bad-sharpe", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 0.2, SortinoRatio: 0.5, MaxDrawdown: 0.10,
		}},
		{Label: "best", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 2.0, SortinoRatio: 3.0, MaxDrawdown: 0.05,
		}},
		{Label: "good", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 1.0, SortinoRatio: 1.5, MaxDrawdown: 0.10,
		}},
		{Label: "bad-drawdown", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 1.5, SortinoRatio: 2.0, MaxDrawdown: 0.50,
		}},
	}

	ranked := FilterAndRank(results, cfg, 10)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 qualified results, got %d", len(ranked))
	}
	if ranked[0].Label != "best" {
		t.Fatalf("expected first = best, got %s", ranked[0].Label)
	}
	if ranked[1].Label != "good" {
		t.Fatalf("expected second = good, got %s", ranked[1].Label)
	}
}

func TestFilterAndRankReturnsTopN(t *testing.T) {
	t.Parallel()
	cfg := DefaultScoringConfig()

	results := []SweepResult{
		{Label: "a", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 2.0, SortinoRatio: 3.0, MaxDrawdown: 0.05,
		}},
		{Label: "b", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 1.5, SortinoRatio: 2.0, MaxDrawdown: 0.10,
		}},
		{Label: "c", Metrics: backtest.Metrics{
			TotalBars: 50, SharpeRatio: 1.0, SortinoRatio: 1.5, MaxDrawdown: 0.10,
		}},
	}

	ranked := FilterAndRank(results, cfg, 1)
	if len(ranked) != 1 {
		t.Fatalf("expected 1 result, got %d", len(ranked))
	}
	if ranked[0].Label != "a" {
		t.Fatalf("expected top = a, got %s", ranked[0].Label)
	}
}
