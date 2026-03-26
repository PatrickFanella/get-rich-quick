package backtest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

func TestGenerateWalkForwardWindows(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	windows, err := generateWalkForwardWindows(start, end, WalkForwardConfig{
		CalibrationMonths: 2,
		TestMonths:        1,
	})
	if err != nil {
		t.Fatalf("generateWalkForwardWindows() error = %v", err)
	}

	if len(windows) != 4 {
		t.Fatalf("len(windows) = %d, want 4", len(windows))
	}

	if !windows[0].CalibrationStart.Equal(start) {
		t.Errorf("windows[0].CalibrationStart = %v, want %v", windows[0].CalibrationStart, start)
	}

	for i := range windows {
		w := windows[i]
		if !w.CalibrationEnd.Before(w.TestStart) {
			t.Errorf("window %d calibration end (%v) must be before test start (%v)", i, w.CalibrationEnd, w.TestStart)
		}
		if !w.CalibrationStart.Before(w.CalibrationEnd) {
			t.Errorf("window %d calibration start (%v) must be before calibration end (%v)", i, w.CalibrationStart, w.CalibrationEnd)
		}
		if !w.TestStart.Before(w.TestEnd) {
			t.Errorf("window %d test start (%v) must be before test end (%v)", i, w.TestStart, w.TestEnd)
		}
		if i == 0 {
			continue
		}
		wantStart := windows[i-1].CalibrationStart.AddDate(0, 1, 0)
		if !w.CalibrationStart.Equal(wantStart) {
			t.Errorf("window %d calibration start = %v, want %v", i, w.CalibrationStart, wantStart)
		}
	}
}

func TestGenerateWalkForwardWindowsRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	testCases := []WalkForwardConfig{
		{CalibrationMonths: 0, TestMonths: 1},
		{CalibrationMonths: 1, TestMonths: 0},
		{CalibrationMonths: 3, TestMonths: 1}, // no full window in range
	}

	for _, tc := range testCases {
		_, err := generateWalkForwardWindows(start, end, tc)
		if err == nil {
			t.Fatalf("expected error for config %+v", tc)
		}
	}
}

func TestOrchestratorRunWalkForwardAggregatesOutOfSample(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	bars := makeDailyBars(start, end)

	cfg := OrchestratorConfig{
		StrategyID:  uuid.New(),
		Ticker:      "AAPL",
		StartDate:   start,
		EndDate:     end,
		InitialCash: 100_000,
		FillConfig: FillConfig{
			Slippage: ProportionalSlippage{BasisPoints: 0},
		},
	}

	orch, err := NewOrchestrator(cfg, bars, makePipeline(), nil)
	if err != nil {
		t.Fatalf("NewOrchestrator() error = %v", err)
	}

	result, err := orch.RunWalkForward(context.Background(), WalkForwardConfig{
		CalibrationMonths: 2,
		TestMonths:        1,
	})
	if err != nil {
		t.Fatalf("RunWalkForward() error = %v", err)
	}

	if len(result.Windows) != 4 {
		t.Fatalf("len(result.Windows) = %d, want 4", len(result.Windows))
	}
	if result.OutOfSample.TotalReturn.N != 4 {
		t.Errorf("OutOfSample.TotalReturn.N = %d, want 4", result.OutOfSample.TotalReturn.N)
	}
	if len(result.OutOfSample.TotalBars.Values) != 4 {
		t.Errorf("len(OutOfSample.TotalBars.Values) = %d, want 4", len(result.OutOfSample.TotalBars.Values))
	}

	for i, windowResult := range result.Windows {
		if windowResult.Calibration == nil {
			t.Fatalf("window %d calibration result is nil", i)
		}
		if windowResult.Test == nil {
			t.Fatalf("window %d test result is nil", i)
		}
		if !windowResult.Window.CalibrationEnd.Before(windowResult.Window.TestStart) {
			t.Errorf("window %d has overlapping calibration/test ranges", i)
		}
	}
}

func makeDailyBars(start, end time.Time) []domain.OHLCV {
	bars := make([]domain.OHLCV, 0)
	for ts := start; !ts.After(end); ts = ts.Add(24 * time.Hour) {
		bars = append(bars, makeBar(ts, 100))
	}
	return bars
}
