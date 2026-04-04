package backtest

import (
	"math"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// ExpiryEvent describes the outcome of an options position at expiry.
type ExpiryEvent struct {
	OCCSymbol       string
	Position        *domain.Position
	UnderlyingPrice float64
	IntrinsicValue  float64
	IsITM           bool
	ExitReason      string // "expired_worthless" or "exercised"
}

// CheckOptionsExpiry checks all options positions for expiry. Returns a list
// of positions that expired on or before the given date, along with their
// settlement values. Only positions with AssetClass == "option" and a non-nil
// Expiry are evaluated. The underlyingPrices map is keyed by the underlying
// ticker (e.g. "AAPL").
func CheckOptionsExpiry(positions map[string]*domain.Position, now time.Time, underlyingPrices map[string]float64) []ExpiryEvent {
	var events []ExpiryEvent

	// Normalise now to date-only for comparison.
	nowDate := truncateToDate(now)

	for occSymbol, pos := range positions {
		if pos.AssetClass != domain.AssetClassOption {
			continue
		}
		if pos.Expiry == nil {
			continue
		}

		expiryDate := truncateToDate(*pos.Expiry)
		if expiryDate.After(nowDate) {
			continue // not yet expired
		}

		// Determine strike and option type. Prefer the fields on the position;
		// fall back to parsing the OCC symbol.
		strike, optType, ok := resolveContractDetails(pos, occSymbol)
		if !ok {
			continue // unparseable; skip
		}

		underlying := pos.UnderlyingTicker
		if underlying == "" {
			// Try to extract from OCC parse.
			if parsed, err := domain.ParseOCC(occSymbol); err == nil {
				underlying = parsed.Underlying
			}
		}

		underlyingPrice, found := underlyingPrices[underlying]
		if !found {
			continue // no price data for underlying; skip
		}

		intrinsic := intrinsicValue(optType, strike, underlyingPrice)
		itm := intrinsic > 0

		reason := "expired_worthless"
		if itm {
			reason = "exercised"
		}

		events = append(events, ExpiryEvent{
			OCCSymbol:       occSymbol,
			Position:        pos,
			UnderlyingPrice: underlyingPrice,
			IntrinsicValue:  intrinsic,
			IsITM:           itm,
			ExitReason:      reason,
		})
	}

	return events
}

// truncateToDate strips the time component, keeping only date in UTC.
func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// intrinsicValue computes max(0, underlying - strike) for calls and
// max(0, strike - underlying) for puts.
func intrinsicValue(optType domain.OptionType, strike, underlyingPrice float64) float64 {
	switch optType {
	case domain.OptionTypeCall:
		return math.Max(0, underlyingPrice-strike)
	case domain.OptionTypePut:
		return math.Max(0, strike-underlyingPrice)
	default:
		return 0
	}
}

// resolveContractDetails extracts the strike and option type from the position
// fields, falling back to OCC symbol parsing.
func resolveContractDetails(pos *domain.Position, occSymbol string) (float64, domain.OptionType, bool) {
	if pos.Strike != nil && pos.OptionType != nil {
		return *pos.Strike, *pos.OptionType, true
	}
	parsed, err := domain.ParseOCC(occSymbol)
	if err != nil {
		return 0, "", false
	}
	return parsed.Strike, parsed.OptionType, true
}
