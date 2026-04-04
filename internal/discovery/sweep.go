package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/agent/rules"
	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// SweepConfig controls how many parameter variants to generate and backtest.
type SweepConfig struct {
	Ticker      string
	MarketType  domain.MarketType
	Bars        []domain.OHLCV
	StartDate   time.Time
	EndDate     time.Time
	InitialCash float64 // default 100000
	Variations  int     // number of variants to test (default 20)
}

// RunSweep generates parameter variants from a base config and backtests each.
// Returns results sorted by score descending.
func RunSweep(ctx context.Context, baseConfig rules.RulesEngineConfig, cfg SweepConfig, scoring ScoringConfig, logger *slog.Logger) ([]SweepResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.InitialCash == 0 {
		cfg.InitialCash = 100_000
	}
	if cfg.Variations == 0 {
		cfg.Variations = 20
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Build variant list: base config first, then random mutations.
	variants := make([]struct {
		label  string
		config rules.RulesEngineConfig
	}, 0, cfg.Variations+1)

	variants = append(variants, struct {
		label  string
		config rules.RulesEngineConfig
	}{label: "base", config: baseConfig})

	for i := 0; i < cfg.Variations; i++ {
		variants = append(variants, struct {
			label  string
			config rules.RulesEngineConfig
		}{
			label:  fmt.Sprintf("variant_%d", i+1),
			config: mutateConfig(baseConfig, rng),
		})
	}

	results := make([]SweepResult, 0, len(variants))
	for _, v := range variants {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("sweep: context cancelled: %w", err)
		}

		pipeline := rules.NewRulesPipeline(
			v.config,
			cfg.Bars,
			cfg.StartDate,
			cfg.InitialCash,
			agent.NoopPersister{},
			nil,
			logger,
		)

		orch, err := backtest.NewOrchestrator(
			backtest.OrchestratorConfig{
				StrategyID:  [16]byte{1}, // placeholder for sweep runs
				Ticker:      cfg.Ticker,
				StartDate:   cfg.StartDate,
				EndDate:     cfg.EndDate,
				InitialCash: cfg.InitialCash,
				FillConfig: backtest.FillConfig{
					Slippage: backtest.ProportionalSlippage{BasisPoints: 5},
				},
			},
			cfg.Bars,
			pipeline,
			logger,
		)
		if err != nil {
			logger.Warn("sweep: failed to create orchestrator",
				slog.String("label", v.label),
				slog.Any("error", err),
			)
			continue
		}

		result, err := orch.Run(ctx)
		if err != nil {
			logger.Warn("sweep: run failed",
				slog.String("label", v.label),
				slog.Any("error", err),
			)
			continue
		}

		score := ScoreMetrics(result.Metrics, scoring)
		results = append(results, SweepResult{
			Label:   v.label,
			Config:  v.config,
			Metrics: result.Metrics,
			Score:   score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// mutateConfig does a deep copy of the config and applies random mutations to
// numeric parameters.
func mutateConfig(base rules.RulesEngineConfig, rng *rand.Rand) rules.RulesEngineConfig {
	cfg := deepCopyConfig(base)

	// Mutate entry condition values.
	for i := range cfg.Entry.Conditions {
		if cfg.Entry.Conditions[i].Value != nil {
			v := *cfg.Entry.Conditions[i].Value
			mutated := v * (0.8 + rng.Float64()*0.4) // [0.8, 1.2]
			cfg.Entry.Conditions[i].Value = &mutated
		}
	}

	// Mutate exit condition values.
	for i := range cfg.Exit.Conditions {
		if cfg.Exit.Conditions[i].Value != nil {
			v := *cfg.Exit.Conditions[i].Value
			mutated := v * (0.8 + rng.Float64()*0.4) // [0.8, 1.2]
			cfg.Exit.Conditions[i].Value = &mutated
		}
	}

	// Mutate stop_loss.
	if cfg.StopLoss.ATRMultiplier != 0 {
		cfg.StopLoss.ATRMultiplier *= 0.7 + rng.Float64()*0.8 // [0.7, 1.5]
	}
	if cfg.StopLoss.Pct != 0 {
		cfg.StopLoss.Pct *= 0.7 + rng.Float64()*0.8 // [0.7, 1.5]
	}

	// Mutate take_profit.
	if cfg.TakeProfit.Ratio != 0 {
		cfg.TakeProfit.Ratio *= 0.7 + rng.Float64()*1.3 // [0.7, 2.0]
	}
	if cfg.TakeProfit.Pct != 0 {
		cfg.TakeProfit.Pct *= 0.7 + rng.Float64()*0.8 // [0.7, 1.5]
	}

	// Mutate position_sizing.fraction_pct.
	fractionChoices := []float64{2, 3, 5, 8, 10}
	cfg.PositionSizing.FractionPct = fractionChoices[rng.Intn(len(fractionChoices))]

	return cfg
}

// deepCopyConfig creates a new RulesEngineConfig with copies of all slice/pointer fields.
func deepCopyConfig(src rules.RulesEngineConfig) rules.RulesEngineConfig {
	dst := src

	// Deep copy entry conditions.
	dst.Entry.Conditions = make([]rules.Condition, len(src.Entry.Conditions))
	for i, c := range src.Entry.Conditions {
		dst.Entry.Conditions[i] = c
		if c.Value != nil {
			v := *c.Value
			dst.Entry.Conditions[i].Value = &v
		}
	}

	// Deep copy exit conditions.
	dst.Exit.Conditions = make([]rules.Condition, len(src.Exit.Conditions))
	for i, c := range src.Exit.Conditions {
		dst.Exit.Conditions[i] = c
		if c.Value != nil {
			v := *c.Value
			dst.Exit.Conditions[i].Value = &v
		}
	}

	// Deep copy filters if present.
	if src.Filters != nil {
		f := *src.Filters
		dst.Filters = &f
	}

	return dst
}
