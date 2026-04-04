package rules

import (
	"strings"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

func validOptionsConfig() *OptionsRulesConfig {
	return &OptionsRulesConfig{
		Version:      1,
		StrategyType: domain.StrategyIronCondor,
		Underlying:   "SPY",
		Entry: ConditionGroup{
			Operator:   "AND",
			Conditions: []Condition{{Field: "iv_rank", Op: "gt", Value: fp(30)}},
		},
		Exit: ConditionGroup{
			Operator:   "OR",
			Conditions: []Condition{{Field: "pnl_pct", Op: "gt", Value: fp(50)}},
		},
		LegSelection: map[string]LegSelector{
			"short_call": {
				OptionType: domain.OptionTypeCall, DeltaTarget: 0.16,
				DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
				Intent: domain.PositionIntentSellToOpen, Ratio: 1,
			},
		},
		PositionSizing: OptionsSizingConfig{Method: "max_risk", MaxRiskUSD: 500},
		Management:     OptionsManagement{CloseAtProfitPct: 50, CloseAtDTE: 7},
	}
}

func TestValidateOptions_Valid(t *testing.T) {
	t.Parallel()
	if err := ValidateOptions(validOptionsConfig()); err != nil {
		t.Fatalf("valid config should pass: %v", err)
	}
}

func TestValidateOptions_Nil(t *testing.T) {
	t.Parallel()
	if err := ValidateOptions(nil); err == nil {
		t.Fatal("nil config should fail")
	}
}

func TestValidateOptions_BadVersion(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.Version = 0
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected version error, got: %v", err)
	}
}

func TestValidateOptions_UnknownStrategy(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.StrategyType = "magic_spread"
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "unknown strategy_type") {
		t.Fatalf("expected strategy error, got: %v", err)
	}
}

func TestValidateOptions_EmptyUnderlying(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.Underlying = ""
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "underlying") {
		t.Fatalf("expected underlying error, got: %v", err)
	}
}

func TestValidateOptions_NoLegs(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.LegSelection = nil
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "at least one leg") {
		t.Fatalf("expected leg_selection error, got: %v", err)
	}
}

func TestValidateOptions_BadLegOptionType(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.LegSelection["bad"] = LegSelector{
		OptionType: "strangle", DeltaTarget: 0.16,
		DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
		Intent: domain.PositionIntentSellToOpen, Ratio: 1,
	}
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "option_type") {
		t.Fatalf("expected option_type error, got: %v", err)
	}
}

func TestValidateOptions_BadDeltaTarget(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.LegSelection["short_call"] = LegSelector{
		OptionType: domain.OptionTypeCall, DeltaTarget: 1.5,
		DTEMin: 20, DTEMax: 50, Side: domain.OrderSideSell,
		Intent: domain.PositionIntentSellToOpen, Ratio: 1,
	}
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "delta_target") {
		t.Fatalf("expected delta_target error, got: %v", err)
	}
}

func TestValidateOptions_BadDTERange(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.LegSelection["short_call"] = LegSelector{
		OptionType: domain.OptionTypeCall, DeltaTarget: 0.16,
		DTEMin: 50, DTEMax: 20, Side: domain.OrderSideSell,
		Intent: domain.PositionIntentSellToOpen, Ratio: 1,
	}
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "dte_max") {
		t.Fatalf("expected DTE range error, got: %v", err)
	}
}

func TestValidateOptions_BadSide(t *testing.T) {
	t.Parallel()
	cfg := validOptionsConfig()
	cfg.LegSelection["short_call"] = LegSelector{
		OptionType: domain.OptionTypeCall, DeltaTarget: 0.16,
		DTEMin: 20, DTEMax: 50, Side: "hold",
		Intent: domain.PositionIntentSellToOpen, Ratio: 1,
	}
	err := ValidateOptions(cfg)
	if err == nil || !strings.Contains(err.Error(), "side") {
		t.Fatalf("expected side error, got: %v", err)
	}
}

func TestValidateOptions_SizingMethods(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		sizing OptionsSizingConfig
		errStr string
	}{
		{"unknown method", OptionsSizingConfig{Method: "magic"}, "unknown method"},
		{"max_risk zero", OptionsSizingConfig{Method: "max_risk"}, "max_risk_usd"},
		{"fixed_contracts zero", OptionsSizingConfig{Method: "fixed_contracts"}, "fixed_contracts"},
		{"premium_budget zero", OptionsSizingConfig{Method: "premium_budget"}, "premium_budget"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := validOptionsConfig()
			cfg.PositionSizing = tc.sizing
			err := ValidateOptions(cfg)
			if err == nil || !strings.Contains(err.Error(), tc.errStr) {
				t.Errorf("expected error containing %q, got: %v", tc.errStr, err)
			}
		})
	}
}

func TestValidateOptions_ManagementBounds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mgmt   OptionsManagement
		errStr string
	}{
		{"profit > 100", OptionsManagement{CloseAtProfitPct: 101}, "close_at_profit_pct"},
		{"profit < 0", OptionsManagement{CloseAtProfitPct: -1}, "close_at_profit_pct"},
		{"stop > 100", OptionsManagement{StopLossPct: 101}, "stop_loss_pct"},
		{"dte < 0", OptionsManagement{CloseAtDTE: -1}, "close_at_dte"},
		{"roll < 0", OptionsManagement{RollAtDTE: -1}, "roll_at_dte"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := validOptionsConfig()
			cfg.Management = tc.mgmt
			err := ValidateOptions(cfg)
			if err == nil || !strings.Contains(err.Error(), tc.errStr) {
				t.Errorf("expected error containing %q, got: %v", tc.errStr, err)
			}
		})
	}
}

func TestValidateOptions_OptionsFieldsInConditions(t *testing.T) {
	t.Parallel()
	// Verify that options-specific fields (iv_rank, atm_iv, etc.) pass
	// condition validation through the KnownFields map.
	cfg := validOptionsConfig()
	cfg.Entry = ConditionGroup{
		Operator: "AND",
		Conditions: []Condition{
			{Field: "iv_rank", Op: "gt", Value: fp(30)},
			{Field: "iv_percentile", Op: "gt", Value: fp(50)},
			{Field: "atm_iv", Op: "lt", Value: fp(0.40)},
			{Field: "put_call_ratio", Op: "gt", Value: fp(0.8)},
		},
	}
	cfg.Exit = ConditionGroup{
		Operator: "OR",
		Conditions: []Condition{
			{Field: "dte", Op: "lte", Value: fp(7)},
			{Field: "pnl_pct", Op: "gt", Value: fp(50)},
		},
	}
	if err := ValidateOptions(cfg); err != nil {
		t.Fatalf("options fields should be valid in conditions: %v", err)
	}
}
