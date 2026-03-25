package execution_test

import (
	"math"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/execution"
)

func TestATRPositionSize(t *testing.T) {
	t.Parallel()

	got := execution.ATRPositionSize(100000, 0.02, 5, 2)
	want := 200.0

	assertFloatClose(t, got, want)
}

func TestKellyPositionSize(t *testing.T) {
	t.Parallel()

	got := execution.KellyPositionSize(100000, 0.60, 2)
	want := 40000.0

	assertFloatClose(t, got, want)
}

func TestFixedFractionalSize(t *testing.T) {
	t.Parallel()

	got := execution.FixedFractionalSize(100000, 0.10, 50)
	want := 200.0

	assertFloatClose(t, got, want)
}

func TestPositionSizingReturnsZeroForInvalidInputs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  float64
	}{
		{
			name: "atr with non-positive account value",
			got:  execution.ATRPositionSize(0, 0.02, 5, 2),
		},
		{
			name: "atr with non-positive risk percent",
			got:  execution.ATRPositionSize(100000, -0.02, 5, 2),
		},
		{
			name: "kelly with out of range win rate",
			got:  execution.KellyPositionSize(100000, 1.2, 2),
		},
		{
			name: "kelly with negative expectation",
			got:  execution.KellyPositionSize(100000, 0.30, 2),
		},
		{
			name: "fixed fractional with non-positive account value",
			got:  execution.FixedFractionalSize(-100000, 0.10, 50),
		},
		{
			name: "fixed fractional with non-positive fraction",
			got:  execution.FixedFractionalSize(100000, 0, 50),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != 0 {
				t.Fatalf("%s = %v, want 0", tc.name, tc.got)
			}
		})
	}
}

func TestCalculatePositionSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method execution.PositionSizingMethod
		params execution.PositionSizingParams
		want   float64
	}{
		{
			name:   "atr",
			method: execution.PositionSizingMethodATR,
			params: execution.PositionSizingParams{
				AccountValue: 100000,
				RiskPct:      0.02,
				ATR:          5,
				Multiplier:   2,
			},
			want: 200,
		},
		{
			name:   "kelly",
			method: execution.PositionSizingMethodKelly,
			params: execution.PositionSizingParams{
				AccountValue: 100000,
				WinRate:      0.60,
				WinLossRatio: 2,
			},
			want: 40000,
		},
		{
			name:   "fixed fractional",
			method: execution.PositionSizingMethodFixedFractional,
			params: execution.PositionSizingParams{
				AccountValue:  100000,
				FractionPct:   0.10,
				PricePerShare: 50,
			},
			want: 200,
		},
		{
			name:   "half kelly",
			method: execution.PositionSizingMethodKelly,
			params: execution.PositionSizingParams{
				AccountValue: 100000,
				WinRate:      0.60,
				WinLossRatio: 2,
				HalfKelly:    true,
			},
			want: 20000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := execution.CalculatePositionSize(tc.method, tc.params)
			assertFloatClose(t, got, tc.want)
		})
	}
}

func assertFloatClose(t *testing.T, got, want float64) {
	t.Helper()

	const tolerance = 1e-9

	if math.Abs(got-want) > tolerance {
		t.Fatalf("got %v, want %v", got, want)
	}
}
