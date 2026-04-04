package discovery

import (
	"math"
	"sort"

	"github.com/PatrickFanella/get-rich-quick/internal/agent/rules"
	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
)

// ScoringConfig defines thresholds and weights for the composite scoring function.
type ScoringConfig struct {
	MinSharpe      float64 // default 0.5
	MaxDrawdown    float64 // default 0.20
	MinTrades      int     // default 10
	SharpeWeight   float64 // default 0.5
	SortinoWeight  float64 // default 0.3
	DrawdownWeight float64 // default 0.2
}

// DefaultScoringConfig returns a ScoringConfig with sensible defaults.
func DefaultScoringConfig() ScoringConfig {
	return ScoringConfig{
		MinSharpe:      0.5,
		MaxDrawdown:    0.20,
		MinTrades:      10,
		SharpeWeight:   0.5,
		SortinoWeight:  0.3,
		DrawdownWeight: 0.2,
	}
}

// SweepResult pairs a strategy configuration with its backtest metrics and
// composite score.
type SweepResult struct {
	Label   string
	Config  rules.RulesEngineConfig
	Metrics backtest.Metrics
	Score   float64
}

// ScoreMetrics computes a composite score from backtest metrics.
// Returns -Inf for strategies that do not meet the minimum thresholds.
func ScoreMetrics(m backtest.Metrics, cfg ScoringConfig) float64 {
	if m.TotalBars < cfg.MinTrades {
		return math.Inf(-1)
	}
	if m.SharpeRatio < cfg.MinSharpe {
		return math.Inf(-1)
	}
	if m.MaxDrawdown > cfg.MaxDrawdown {
		return math.Inf(-1)
	}
	if math.IsNaN(m.SharpeRatio) || math.IsInf(m.SharpeRatio, 0) ||
		math.IsNaN(m.SortinoRatio) || math.IsInf(m.SortinoRatio, 0) ||
		math.IsNaN(m.MaxDrawdown) || math.IsInf(m.MaxDrawdown, 0) {
		return math.Inf(-1)
	}

	return cfg.SharpeWeight*m.SharpeRatio +
		cfg.SortinoWeight*m.SortinoRatio -
		cfg.DrawdownWeight*m.MaxDrawdown
}

// FilterAndRank removes disqualified results (score == -Inf), sorts the
// remainder by score descending, and returns the top N.
func FilterAndRank(results []SweepResult, cfg ScoringConfig, topN int) []SweepResult {
	var qualified []SweepResult
	for _, r := range results {
		score := ScoreMetrics(r.Metrics, cfg)
		r.Score = score
		if !math.IsInf(score, -1) {
			qualified = append(qualified, r)
		}
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].Score > qualified[j].Score
	})

	if topN > 0 && len(qualified) > topN {
		qualified = qualified[:topN]
	}
	return qualified
}
