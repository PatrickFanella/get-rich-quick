package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/agent/rules"
	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// ValidationConfig controls the walk-forward out-of-sample test parameters.
type ValidationConfig struct {
	CalibrationMonths int     // default 6
	TestMonths        int     // default 3
	MinOOSRatio       float64 // min OOS Sharpe / in-sample Sharpe (default 0.5)
}

// ValidationResult holds the outcome of a walk-forward validation run.
type ValidationResult struct {
	Passed      bool
	InSample    backtest.Metrics
	OutOfSample backtest.Metrics
	OOSRatio    float64 // OOS Sharpe / in-sample Sharpe
	Reason      string
}

// ValidateOutOfSample runs walk-forward analysis on the discovered strategy
// and checks if out-of-sample performance meets the threshold.
func ValidateOutOfSample(
	ctx context.Context,
	cfg ValidationConfig,
	bars []domain.OHLCV,
	rulesConfig rules.RulesEngineConfig,
	startDate, endDate time.Time,
	initialCash float64,
	logger *slog.Logger,
) (*ValidationResult, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.CalibrationMonths == 0 {
		cfg.CalibrationMonths = 6
	}
	if cfg.TestMonths == 0 {
		cfg.TestMonths = 3
	}
	if cfg.MinOOSRatio == 0 {
		cfg.MinOOSRatio = 0.5
	}

	pipeline := rules.NewRulesPipeline(
		rulesConfig,
		bars,
		startDate,
		initialCash,
		agent.NoopPersister{},
		nil,
		logger,
	)

	orch, err := backtest.NewOrchestrator(
		backtest.OrchestratorConfig{
			StrategyID:  [16]byte{2}, // placeholder for validation runs
			Ticker:      "validation",
			StartDate:   startDate,
			EndDate:     endDate,
			InitialCash: initialCash,
			FillConfig: backtest.FillConfig{
				Slippage: backtest.ProportionalSlippage{BasisPoints: 5},
			},
		},
		bars,
		pipeline,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("validator: creating orchestrator: %w", err)
	}

	wfResult, err := orch.RunWalkForward(ctx, backtest.WalkForwardConfig{
		CalibrationMonths: cfg.CalibrationMonths,
		TestMonths:        cfg.TestMonths,
	})
	if err != nil {
		return nil, fmt.Errorf("validator: walk-forward failed: %w", err)
	}

	if len(wfResult.Windows) == 0 {
		return nil, fmt.Errorf("validator: no walk-forward windows produced")
	}

	// In-sample metrics come from the first calibration window.
	inSample := wfResult.Windows[0].Calibration.Metrics
	inSampleSharpe := inSample.SharpeRatio

	// OOS metrics are aggregated from all test windows.
	oosSharpe := wfResult.OutOfSample.SharpeRatio.Mean

	// Build a synthetic Metrics from the aggregated OOS stats for reporting.
	oosMetrics := backtest.Metrics{
		TotalReturn:  wfResult.OutOfSample.TotalReturn.Mean,
		SharpeRatio:  oosSharpe,
		SortinoRatio: wfResult.OutOfSample.SortinoRatio.Mean,
		MaxDrawdown:  wfResult.OutOfSample.MaxDrawdown.Mean,
	}

	result := &ValidationResult{
		InSample:    inSample,
		OutOfSample: oosMetrics,
	}

	// Compute OOS ratio.
	if inSampleSharpe != 0 {
		result.OOSRatio = oosSharpe / inSampleSharpe
	}

	// Check thresholds.
	if oosSharpe < 0 {
		result.Passed = false
		result.Reason = fmt.Sprintf("OOS Sharpe negative (%.4f)", oosSharpe)
		return result, nil
	}

	if result.OOSRatio < cfg.MinOOSRatio {
		result.Passed = false
		result.Reason = fmt.Sprintf(
			"OOS ratio %.4f below minimum %.4f (OOS Sharpe=%.4f, in-sample Sharpe=%.4f)",
			result.OOSRatio, cfg.MinOOSRatio, oosSharpe, inSampleSharpe,
		)
		return result, nil
	}

	result.Passed = true
	return result, nil
}
