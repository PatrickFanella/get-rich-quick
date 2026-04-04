package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// optTypePtr is a helper to create *domain.OptionType inline.
func optTypePtr(ot domain.OptionType) *domain.OptionType { return &ot }

func TestCheckOptionsExpiry_ITMCallExercised(t *testing.T) {
	t.Parallel()
	expiry := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	positions := map[string]*domain.Position{
		"AAPL260417C00150000": {
			Ticker:           "AAPL260417C00150000",
			AssetClass:       domain.AssetClassOption,
			UnderlyingTicker: "AAPL",
			OptionType:       optTypePtr(domain.OptionTypeCall),
			Strike:           floatPtr(150),
			Expiry:           timePtr(expiry),
			Side:             domain.PositionSideLong,
			Quantity:         2,
			AvgEntry:         5.0,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC)
	underlying := map[string]float64{"AAPL": 175.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if !ev.IsITM {
		t.Error("expected ITM")
	}
	if ev.ExitReason != "exercised" {
		t.Errorf("ExitReason = %q, want %q", ev.ExitReason, "exercised")
	}
	// Intrinsic = 175 - 150 = 25
	if math.Abs(ev.IntrinsicValue-25.0) > 1e-9 {
		t.Errorf("IntrinsicValue = %v, want 25", ev.IntrinsicValue)
	}
	if ev.UnderlyingPrice != 175.0 {
		t.Errorf("UnderlyingPrice = %v, want 175", ev.UnderlyingPrice)
	}
}

func TestCheckOptionsExpiry_OTMPutExpiresWorthless(t *testing.T) {
	t.Parallel()
	expiry := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	positions := map[string]*domain.Position{
		"AAPL260417P00130000": {
			Ticker:           "AAPL260417P00130000",
			AssetClass:       domain.AssetClassOption,
			UnderlyingTicker: "AAPL",
			OptionType:       optTypePtr(domain.OptionTypePut),
			Strike:           floatPtr(130),
			Expiry:           timePtr(expiry),
			Side:             domain.PositionSideLong,
			Quantity:         1,
			AvgEntry:         2.0,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC)
	underlying := map[string]float64{"AAPL": 175.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.IsITM {
		t.Error("expected OTM")
	}
	if ev.ExitReason != "expired_worthless" {
		t.Errorf("ExitReason = %q, want %q", ev.ExitReason, "expired_worthless")
	}
	if ev.IntrinsicValue != 0 {
		t.Errorf("IntrinsicValue = %v, want 0", ev.IntrinsicValue)
	}
}

func TestCheckOptionsExpiry_NonOptionIgnored(t *testing.T) {
	t.Parallel()
	positions := map[string]*domain.Position{
		"AAPL": {
			Ticker:     "AAPL",
			AssetClass: domain.AssetClassEquity,
			Side:       domain.PositionSideLong,
			Quantity:   100,
			AvgEntry:   150,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC)
	underlying := map[string]float64{"AAPL": 175.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for equity position, got %d", len(events))
	}
}

func TestCheckOptionsExpiry_NilExpiryIgnored(t *testing.T) {
	t.Parallel()
	positions := map[string]*domain.Position{
		"AAPL260417C00150000": {
			Ticker:           "AAPL260417C00150000",
			AssetClass:       domain.AssetClassOption,
			UnderlyingTicker: "AAPL",
			OptionType:       optTypePtr(domain.OptionTypeCall),
			Strike:           floatPtr(150),
			Expiry:           nil, // no expiry set
			Side:             domain.PositionSideLong,
			Quantity:         1,
			AvgEntry:         5.0,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC)
	underlying := map[string]float64{"AAPL": 175.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for position without expiry, got %d", len(events))
	}
}

func TestCheckOptionsExpiry_NotYetExpired(t *testing.T) {
	t.Parallel()
	expiry := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC) // future
	positions := map[string]*domain.Position{
		"AAPL260515C00150000": {
			Ticker:           "AAPL260515C00150000",
			AssetClass:       domain.AssetClassOption,
			UnderlyingTicker: "AAPL",
			OptionType:       optTypePtr(domain.OptionTypeCall),
			Strike:           floatPtr(150),
			Expiry:           timePtr(expiry),
			Side:             domain.PositionSideLong,
			Quantity:         1,
			AvgEntry:         5.0,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC) // before expiry
	underlying := map[string]float64{"AAPL": 175.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 0 {
		t.Fatalf("expected 0 events for not-yet-expired option, got %d", len(events))
	}
}

func TestCheckOptionsExpiry_FallsBackToOCCParsing(t *testing.T) {
	t.Parallel()
	expiry := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
	positions := map[string]*domain.Position{
		"SPY260417P00400000": {
			Ticker:           "SPY260417P00400000",
			AssetClass:       domain.AssetClassOption,
			UnderlyingTicker: "SPY",
			// OptionType and Strike NOT set -- force OCC fallback
			Expiry:   timePtr(expiry),
			Side:     domain.PositionSideLong,
			Quantity: 3,
			AvgEntry: 8.0,
		},
	}
	now := time.Date(2026, 4, 17, 16, 0, 0, 0, time.UTC)
	underlying := map[string]float64{"SPY": 450.0}

	events := CheckOptionsExpiry(positions, now, underlying)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	// SPY at 450, put strike 400 -> OTM
	if ev.IsITM {
		t.Error("expected OTM for put with strike 400 and underlying 450")
	}
	if ev.ExitReason != "expired_worthless" {
		t.Errorf("ExitReason = %q, want %q", ev.ExitReason, "expired_worthless")
	}
}
