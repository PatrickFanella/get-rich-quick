package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

func TestPositionTrackerTracksCostBasisAndUnrealizedPnL(t *testing.T) {
	tracker, err := NewPositionTracker(1000)
	if err != nil {
		t.Fatalf("NewPositionTracker() error = %v", err)
	}

	ts := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	err = tracker.ApplyTrade(domain.Trade{
		Ticker:     "aapl",
		Side:       domain.OrderSideBuy,
		Quantity:   2,
		Price:      100,
		Fee:        2,
		ExecutedAt: ts,
	})
	if err != nil {
		t.Fatalf("ApplyTrade() error = %v", err)
	}
	if err := tracker.UpdateMarketPrice("AAPL", 110); err != nil {
		t.Fatalf("UpdateMarketPrice() error = %v", err)
	}

	point := tracker.RecordEquity(ts)
	positions := tracker.Positions()
	if len(positions) != 1 {
		t.Fatalf("len(Positions()) = %d, want 1", len(positions))
	}

	position := positions[0]
	if position.Ticker != "AAPL" {
		t.Fatalf("position.Ticker = %q, want %q", position.Ticker, "AAPL")
	}
	assertFloatEqual(t, position.AvgEntry, 101, "position.AvgEntry")
	assertFloatEqual(t, position.CostBasis, 202, "position.CostBasis")
	assertFloatEqual(t, position.MarketValue, 220, "position.MarketValue")
	assertFloatEqual(t, position.UnrealizedPnL, 18, "position.UnrealizedPnL")
	assertFloatEqual(t, point.Cash, 798, "point.Cash")
	assertFloatEqual(t, point.MarketValue, 220, "point.MarketValue")
	assertFloatEqual(t, point.RealizedPnL, 0, "point.RealizedPnL")
	assertFloatEqual(t, point.UnrealizedPnL, 18, "point.UnrealizedPnL")
	assertFloatEqual(t, point.Equity, 1018, "point.Equity")
	assertFloatEqual(t, point.TotalPnL, 18, "point.TotalPnL")
}

func TestPositionTrackerAggregatesAcrossMultiplePositionsAndBars(t *testing.T) {
	tracker, err := NewPositionTracker(10000)
	if err != nil {
		t.Fatalf("NewPositionTracker() error = %v", err)
	}

	t1 := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	for _, trade := range []domain.Trade{
		{Ticker: "AAPL", Side: domain.OrderSideBuy, Quantity: 10, Price: 100, ExecutedAt: t1},
		{Ticker: "MSFT", Side: domain.OrderSideBuy, Quantity: 5, Price: 200, Fee: 5, ExecutedAt: t1},
	} {
		if err := tracker.ApplyTrade(trade); err != nil {
			t.Fatalf("ApplyTrade(%+v) error = %v", trade, err)
		}
	}

	if err := tracker.UpdateMarketPrice("AAPL", 105); err != nil {
		t.Fatalf("UpdateMarketPrice(AAPL) error = %v", err)
	}
	if err := tracker.UpdateMarketPrice("MSFT", 190); err != nil {
		t.Fatalf("UpdateMarketPrice(MSFT) error = %v", err)
	}

	point1 := tracker.RecordEquity(t1)
	assertFloatEqual(t, point1.Cash, 7995, "point1.Cash")
	assertFloatEqual(t, point1.MarketValue, 2000, "point1.MarketValue")
	assertFloatEqual(t, point1.RealizedPnL, 0, "point1.RealizedPnL")
	assertFloatEqual(t, point1.UnrealizedPnL, -5, "point1.UnrealizedPnL")
	assertFloatEqual(t, point1.Equity, 9995, "point1.Equity")
	assertFloatEqual(t, point1.TotalPnL, -5, "point1.TotalPnL")

	if err := tracker.ApplyTrade(domain.Trade{
		Ticker:     "AAPL",
		Side:       domain.OrderSideSell,
		Quantity:   5,
		Price:      115,
		ExecutedAt: t2,
	}); err != nil {
		t.Fatalf("ApplyTrade(AAPL sell) error = %v", err)
	}
	if err := tracker.UpdateMarketPrice("AAPL", 112); err != nil {
		t.Fatalf("UpdateMarketPrice(AAPL second bar) error = %v", err)
	}
	if err := tracker.UpdateMarketPrice("MSFT", 210); err != nil {
		t.Fatalf("UpdateMarketPrice(MSFT second bar) error = %v", err)
	}

	point2 := tracker.RecordEquity(t2)
	positions := tracker.Positions()
	if len(positions) != 2 {
		t.Fatalf("len(Positions()) = %d, want 2", len(positions))
	}
	if positions[0].Ticker != "AAPL" || positions[1].Ticker != "MSFT" {
		t.Fatalf("Positions() tickers = [%q, %q], want [AAPL, MSFT]", positions[0].Ticker, positions[1].Ticker)
	}

	assertFloatEqual(t, point2.Cash, 8570, "point2.Cash")
	assertFloatEqual(t, point2.MarketValue, 1610, "point2.MarketValue")
	assertFloatEqual(t, point2.RealizedPnL, 75, "point2.RealizedPnL")
	assertFloatEqual(t, point2.UnrealizedPnL, 105, "point2.UnrealizedPnL")
	assertFloatEqual(t, point2.Equity, 10180, "point2.Equity")
	assertFloatEqual(t, point2.TotalPnL, 180, "point2.TotalPnL")

	curve := tracker.EquityCurve()
	if len(curve) != 2 {
		t.Fatalf("len(EquityCurve()) = %d, want 2", len(curve))
	}
	if !curve[0].Timestamp.Equal(t1) || !curve[1].Timestamp.Equal(t2) {
		t.Fatalf("EquityCurve() timestamps = [%s, %s], want [%s, %s]", curve[0].Timestamp, curve[1].Timestamp, t1, t2)
	}
}

func TestPositionTrackerHandlesReversalAndAllocatesFees(t *testing.T) {
	tracker, err := NewPositionTracker(1000)
	if err != nil {
		t.Fatalf("NewPositionTracker() error = %v", err)
	}

	t1 := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	if err := tracker.ApplyTrade(domain.Trade{
		Ticker:     "AAPL",
		Side:       domain.OrderSideBuy,
		Quantity:   2,
		Price:      100,
		Fee:        2,
		ExecutedAt: t1,
	}); err != nil {
		t.Fatalf("ApplyTrade(open long) error = %v", err)
	}
	if err := tracker.ApplyTrade(domain.Trade{
		Ticker:     "AAPL",
		Side:       domain.OrderSideSell,
		Quantity:   3,
		Price:      110,
		Fee:        3,
		ExecutedAt: t2,
	}); err != nil {
		t.Fatalf("ApplyTrade(reverse short) error = %v", err)
	}
	if err := tracker.UpdateMarketPrice("AAPL", 105); err != nil {
		t.Fatalf("UpdateMarketPrice() error = %v", err)
	}

	point := tracker.RecordEquity(t2)
	positions := tracker.Positions()
	if len(positions) != 1 {
		t.Fatalf("len(Positions()) = %d, want 1", len(positions))
	}

	position := positions[0]
	if position.Side != domain.PositionSideShort {
		t.Fatalf("position.Side = %q, want %q", position.Side, domain.PositionSideShort)
	}
	assertFloatEqual(t, position.Quantity, 1, "position.Quantity")
	assertFloatEqual(t, position.AvgEntry, 109, "position.AvgEntry")
	assertFloatEqual(t, position.RealizedPnL, 16, "position.RealizedPnL")
	assertFloatEqual(t, position.UnrealizedPnL, 4, "position.UnrealizedPnL")
	assertFloatEqual(t, point.Cash, 1125, "point.Cash")
	assertFloatEqual(t, point.RealizedPnL, 16, "point.RealizedPnL")
	assertFloatEqual(t, point.UnrealizedPnL, 4, "point.UnrealizedPnL")
	assertFloatEqual(t, point.Equity, 1020, "point.Equity")
	assertFloatEqual(t, point.TotalPnL, 20, "point.TotalPnL")
}

func assertFloatEqual(t *testing.T, got float64, want float64, label string) {
	t.Helper()

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %f, want %f", label, got, want)
	}
}
