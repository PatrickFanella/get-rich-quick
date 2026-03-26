package backtest

import (
	"context"
	"fmt"
	"time"
)

// WalkForwardConfig configures rolling walk-forward analysis.
type WalkForwardConfig struct {
	CalibrationMonths int
	TestMonths        int
}

// WalkForwardWindow defines one calibration/test partition.
type WalkForwardWindow struct {
	CalibrationStart time.Time
	CalibrationEnd   time.Time
	TestStart        time.Time
	TestEnd          time.Time
}

// WalkForwardWindowResult contains per-window run results.
type WalkForwardWindowResult struct {
	Window      WalkForwardWindow
	Calibration *OrchestratorResult
	Test        *OrchestratorResult
}

// WalkForwardResult contains all window results and aggregated out-of-sample metrics.
type WalkForwardResult struct {
	Config      WalkForwardConfig
	Windows     []WalkForwardWindowResult
	OutOfSample AggregatedMetrics
}

// RunWalkForward executes rolling walk-forward analysis by calibrating on N
// months and testing on the subsequent M months, then advancing by M months.
func (o *Orchestrator) RunWalkForward(ctx context.Context, cfg WalkForwardConfig) (*WalkForwardResult, error) {
	windows, err := generateWalkForwardWindows(o.config.StartDate, o.config.EndDate, cfg)
	if err != nil {
		return nil, err
	}

	results := make([]WalkForwardWindowResult, 0, len(windows))
	testMetrics := make([]Metrics, 0, len(windows))

	for i, window := range windows {
		calibrationCfg := o.config
		calibrationCfg.StartDate = window.CalibrationStart
		calibrationCfg.EndDate = window.CalibrationEnd

		calibrationOrchestrator, err := NewOrchestrator(
			calibrationCfg,
			o.bars,
			o.pipeline,
			o.logger,
			o.clockTargets...,
		)
		if err != nil {
			return nil, fmt.Errorf("backtest: creating calibration orchestrator for window %d: %w", i+1, err)
		}

		calibrationResult, err := calibrationOrchestrator.Run(ctx)
		if err != nil {
			return nil, fmt.Errorf("backtest: calibration run failed for window %d: %w", i+1, err)
		}

		testCfg := o.config
		testCfg.StartDate = window.TestStart
		testCfg.EndDate = window.TestEnd

		testOrchestrator, err := NewOrchestrator(
			testCfg,
			o.bars,
			o.pipeline,
			o.logger,
			o.clockTargets...,
		)
		if err != nil {
			return nil, fmt.Errorf("backtest: creating test orchestrator for window %d: %w", i+1, err)
		}

		testResult, err := testOrchestrator.Run(ctx)
		if err != nil {
			return nil, fmt.Errorf("backtest: test run failed for window %d: %w", i+1, err)
		}

		results = append(results, WalkForwardWindowResult{
			Window:      window,
			Calibration: calibrationResult,
			Test:        testResult,
		})
		testMetrics = append(testMetrics, testResult.Metrics)
	}

	return &WalkForwardResult{
		Config:      cfg,
		Windows:     results,
		OutOfSample: aggregateMetrics(testMetrics),
	}, nil
}

func generateWalkForwardWindows(start, end time.Time, cfg WalkForwardConfig) ([]WalkForwardWindow, error) {
	if cfg.CalibrationMonths <= 0 {
		return nil, fmt.Errorf("backtest: calibration months must be > 0")
	}
	if cfg.TestMonths <= 0 {
		return nil, fmt.Errorf("backtest: test months must be > 0")
	}
	if start.IsZero() {
		return nil, fmt.Errorf("backtest: start date is required")
	}
	if end.IsZero() {
		return nil, fmt.Errorf("backtest: end date is required")
	}
	if end.Before(start) {
		return nil, fmt.Errorf("backtest: end date must not be before start date")
	}

	endExclusive := end.Add(time.Nanosecond)
	cursor := start
	windows := make([]WalkForwardWindow, 0)

	for {
		calibrationEndExclusive := cursor.AddDate(0, cfg.CalibrationMonths, 0)
		testStart := calibrationEndExclusive
		testEndExclusive := testStart.AddDate(0, cfg.TestMonths, 0)

		if testEndExclusive.After(endExclusive) {
			break
		}

		windows = append(windows, WalkForwardWindow{
			CalibrationStart: cursor,
			CalibrationEnd:   calibrationEndExclusive.Add(-time.Nanosecond),
			TestStart:        testStart,
			TestEnd:          testEndExclusive.Add(-time.Nanosecond),
		})

		cursor = cursor.AddDate(0, cfg.TestMonths, 0)
	}

	if len(windows) == 0 {
		return nil, fmt.Errorf(
			"backtest: no valid walk-forward windows for range %s to %s with calibration=%d months and test=%d months",
			start.Format(time.DateOnly),
			end.Format(time.DateOnly),
			cfg.CalibrationMonths,
			cfg.TestMonths,
		)
	}

	return windows, nil
}
