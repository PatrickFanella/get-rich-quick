package rules

import (
	"strings"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// mockChain builds a small option chain around the given underlying price.
func mockChain(now time.Time) []domain.OptionSnapshot {
	exp30 := now.AddDate(0, 0, 30)
	exp45 := now.AddDate(0, 0, 45)
	exp10 := now.AddDate(0, 0, 10) // outside typical 20-50 DTE range

	return []domain.OptionSnapshot{
		// 30 DTE calls
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501C00400000", Underlying: "SPY", OptionType: domain.OptionTypeCall, Strike: 400, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: 0.50, IV: 0.20}, Bid: 5, Ask: 5.2, Mid: 5.1, Volume: 100},
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501C00410000", Underlying: "SPY", OptionType: domain.OptionTypeCall, Strike: 410, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: 0.30, IV: 0.22}, Bid: 3, Ask: 3.2, Mid: 3.1, Volume: 80},
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501C00420000", Underlying: "SPY", OptionType: domain.OptionTypeCall, Strike: 420, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: 0.16, IV: 0.24}, Bid: 1.5, Ask: 1.7, Mid: 1.6, Volume: 60},
		// 30 DTE puts
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501P00400000", Underlying: "SPY", OptionType: domain.OptionTypePut, Strike: 400, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: -0.50, IV: 0.20}, Bid: 5, Ask: 5.2, Mid: 5.1, Volume: 120},
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501P00390000", Underlying: "SPY", OptionType: domain.OptionTypePut, Strike: 390, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: -0.30, IV: 0.22}, Bid: 3, Ask: 3.2, Mid: 3.1, Volume: 90},
		{Contract: domain.OptionContract{OCCSymbol: "SPY250501P00380000", Underlying: "SPY", OptionType: domain.OptionTypePut, Strike: 380, Expiry: exp30, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: -0.16, IV: 0.24}, Bid: 1.5, Ask: 1.7, Mid: 1.6, Volume: 50},
		// 45 DTE call (different expiry)
		{Contract: domain.OptionContract{OCCSymbol: "SPY250515C00420000", Underlying: "SPY", OptionType: domain.OptionTypeCall, Strike: 420, Expiry: exp45, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: 0.18, IV: 0.25}, Bid: 2, Ask: 2.2, Mid: 2.1, Volume: 40},
		// 10 DTE call (short-dated, often filtered out)
		{Contract: domain.OptionContract{OCCSymbol: "SPY250411C00410000", Underlying: "SPY", OptionType: domain.OptionTypeCall, Strike: 410, Expiry: exp10, Multiplier: 100}, Greeks: domain.OptionGreeks{Delta: 0.15, IV: 0.30}, Bid: 0.5, Ask: 0.7, Mid: 0.6, Volume: 20},
	}
}

func TestSelectLeg_ClosestDelta(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	sel := LegSelector{
		OptionType:  domain.OptionTypeCall,
		DeltaTarget: 0.16,
		DTEMin:      20,
		DTEMax:      50,
		Side:        domain.OrderSideSell,
		Intent:      domain.PositionIntentSellToOpen,
		Ratio:       1,
	}

	snap, err := SelectLeg(chain, sel, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Contract.Strike != 420 || snap.Contract.Expiry.Equal(now.AddDate(0, 0, 30)) == false {
		t.Errorf("expected 420 strike 30-DTE call, got strike=%v expiry=%v",
			snap.Contract.Strike, snap.Contract.Expiry)
	}
	if snap.Greeks.Delta != 0.16 {
		t.Errorf("expected delta=0.16, got %v", snap.Greeks.Delta)
	}
}

func TestSelectLeg_RespectsDTERange(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	// Only allow 40-50 DTE: should pick the 45-DTE call.
	sel := LegSelector{
		OptionType:  domain.OptionTypeCall,
		DeltaTarget: 0.16,
		DTEMin:      40,
		DTEMax:      50,
		Side:        domain.OrderSideSell,
		Intent:      domain.PositionIntentSellToOpen,
		Ratio:       1,
	}

	snap, err := SelectLeg(chain, sel, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The only 40-50 DTE call has delta 0.18 at strike 420.
	if snap.Greeks.Delta != 0.18 {
		t.Errorf("expected delta=0.18 (45-DTE call), got %v", snap.Greeks.Delta)
	}
}

func TestSelectLeg_EmptyChain(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)

	sel := LegSelector{
		OptionType:  domain.OptionTypeCall,
		DeltaTarget: 0.30,
		DTEMin:      20,
		DTEMax:      50,
		Side:        domain.OrderSideBuy,
		Intent:      domain.PositionIntentBuyToOpen,
		Ratio:       1,
	}

	_, err := SelectLeg(nil, sel, now)
	if err == nil {
		t.Fatal("expected error on empty chain")
	}
	if !strings.Contains(err.Error(), "empty chain") {
		t.Errorf("expected 'empty chain' error, got: %v", err)
	}
}

func TestSelectLeg_NoMatchingContracts(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	// Impossible DTE range.
	sel := LegSelector{
		OptionType:  domain.OptionTypeCall,
		DeltaTarget: 0.30,
		DTEMin:      100,
		DTEMax:      200,
		Side:        domain.OrderSideBuy,
		Intent:      domain.PositionIntentBuyToOpen,
		Ratio:       1,
	}

	_, err := SelectLeg(chain, sel, now)
	if err == nil {
		t.Fatal("expected error when no contracts match DTE range")
	}
	if !strings.Contains(err.Error(), "no contracts match") {
		t.Errorf("expected 'no contracts match' error, got: %v", err)
	}
}

func TestSelectLeg_PutLeg(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	sel := LegSelector{
		OptionType:  domain.OptionTypePut,
		DeltaTarget: 0.16,
		DTEMin:      20,
		DTEMax:      50,
		Side:        domain.OrderSideSell,
		Intent:      domain.PositionIntentSellToOpen,
		Ratio:       1,
	}

	snap, err := SelectLeg(chain, sel, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Contract.OptionType != domain.OptionTypePut {
		t.Errorf("expected put, got %v", snap.Contract.OptionType)
	}
	if snap.Contract.Strike != 380 {
		t.Errorf("expected strike=380 (delta -0.16), got %v", snap.Contract.Strike)
	}
}

func TestSelectSpreadLegs_IronCondor(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	selectors := map[string]LegSelector{
		"short_call": {
			OptionType: domain.OptionTypeCall, DeltaTarget: 0.16,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
			Intent: domain.PositionIntentSellToOpen, Ratio: 1,
		},
		"long_call": {
			OptionType: domain.OptionTypeCall, DeltaTarget: 0.30,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideBuy,
			Intent: domain.PositionIntentBuyToOpen, Ratio: 1,
		},
		"short_put": {
			OptionType: domain.OptionTypePut, DeltaTarget: 0.16,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
			Intent: domain.PositionIntentSellToOpen, Ratio: 1,
		},
		"long_put": {
			OptionType: domain.OptionTypePut, DeltaTarget: 0.30,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideBuy,
			Intent: domain.PositionIntentBuyToOpen, Ratio: 1,
		},
	}

	legs, err := SelectSpreadLegs(chain, selectors, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(legs) != 4 {
		t.Fatalf("expected 4 legs, got %d", len(legs))
	}

	// Verify each leg was selected.
	for _, name := range []string{"short_call", "long_call", "short_put", "long_put"} {
		if legs[name] == nil {
			t.Errorf("leg %q is nil", name)
		}
	}

	// Short call should be the 0.16 delta call.
	if legs["short_call"].Contract.Strike != 420 {
		t.Errorf("short_call: expected strike=420, got %v", legs["short_call"].Contract.Strike)
	}
	// Long call should be the 0.30 delta call.
	if legs["long_call"].Contract.Strike != 410 {
		t.Errorf("long_call: expected strike=410, got %v", legs["long_call"].Contract.Strike)
	}
	// Short put should be the 0.16 delta put.
	if legs["short_put"].Contract.Strike != 380 {
		t.Errorf("short_put: expected strike=380, got %v", legs["short_put"].Contract.Strike)
	}
	// Long put should be the 0.30 delta put.
	if legs["long_put"].Contract.Strike != 390 {
		t.Errorf("long_put: expected strike=390, got %v", legs["long_put"].Contract.Strike)
	}
}

func TestBuildSpread(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)
	chain := mockChain(now)

	selectors := map[string]LegSelector{
		"short_put": {
			OptionType: domain.OptionTypePut, DeltaTarget: 0.30,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
			Intent: domain.PositionIntentSellToOpen, Ratio: 1,
		},
		"long_put": {
			OptionType: domain.OptionTypePut, DeltaTarget: 0.16,
			DTEMin: 20, DTEMax: 50, Side: domain.OrderSideBuy,
			Intent: domain.PositionIntentBuyToOpen, Ratio: 1,
		},
	}

	legs, err := SelectSpreadLegs(chain, selectors, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spread, err := BuildSpread(domain.StrategyBullPutSpread, "SPY", legs, selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spread.StrategyType != domain.StrategyBullPutSpread {
		t.Errorf("expected strategy_type=%s, got %s", domain.StrategyBullPutSpread, spread.StrategyType)
	}
	if spread.Underlying != "SPY" {
		t.Errorf("expected underlying=SPY, got %s", spread.Underlying)
	}
	if len(spread.Legs) != 2 {
		t.Fatalf("expected 2 legs, got %d", len(spread.Legs))
	}
}

func TestBuildSpread_NoLegs(t *testing.T) {
	t.Parallel()
	_, err := BuildSpread(domain.StrategyIronCondor, "SPY", nil, nil)
	if err == nil {
		t.Fatal("expected error on empty legs")
	}
}
