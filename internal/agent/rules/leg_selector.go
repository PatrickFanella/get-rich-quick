package rules

import (
	"fmt"
	"math"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// SelectLeg picks the contract from the chain that best matches the selector
// criteria. It filters by option type and DTE range, then returns the contract
// whose absolute delta is closest to the target.
func SelectLeg(chain []domain.OptionSnapshot, selector LegSelector, now time.Time) (*domain.OptionSnapshot, error) {
	if len(chain) == 0 {
		return nil, fmt.Errorf("leg_selector: empty chain")
	}

	var candidates []domain.OptionSnapshot

	for _, snap := range chain {
		// Filter by option type.
		if snap.Contract.OptionType != selector.OptionType {
			continue
		}
		// Filter by DTE range.
		dte := daysToExpiry(snap.Contract.Expiry, now)
		if dte < selector.DTEMin || dte > selector.DTEMax {
			continue
		}
		candidates = append(candidates, snap)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("leg_selector: no contracts match option_type=%s dte=[%d,%d]",
			selector.OptionType, selector.DTEMin, selector.DTEMax)
	}

	// Find the candidate whose |delta| is closest to the target.
	best := candidates[0]
	bestDist := math.Abs(math.Abs(best.Greeks.Delta) - math.Abs(selector.DeltaTarget))

	for _, c := range candidates[1:] {
		dist := math.Abs(math.Abs(c.Greeks.Delta) - math.Abs(selector.DeltaTarget))
		if dist < bestDist {
			bestDist = dist
			best = c
		}
	}

	return &best, nil
}

// SelectSpreadLegs selects all legs for a spread strategy from the chain.
func SelectSpreadLegs(chain []domain.OptionSnapshot, selectors map[string]LegSelector, now time.Time) (map[string]*domain.OptionSnapshot, error) {
	if len(selectors) == 0 {
		return nil, fmt.Errorf("leg_selector: no leg selectors provided")
	}

	result := make(map[string]*domain.OptionSnapshot, len(selectors))
	for name, sel := range selectors {
		snap, err := SelectLeg(chain, sel, now)
		if err != nil {
			return nil, fmt.Errorf("leg_selector: leg %q: %w", name, err)
		}
		result[name] = snap
	}
	return result, nil
}

// BuildSpread constructs an OptionSpread from selected legs.
func BuildSpread(
	strategyType domain.OptionStrategyType,
	underlying string,
	selectedLegs map[string]*domain.OptionSnapshot,
	selectors map[string]LegSelector,
) (*domain.OptionSpread, error) {
	if len(selectedLegs) == 0 {
		return nil, fmt.Errorf("leg_selector: no selected legs")
	}

	spread := &domain.OptionSpread{
		StrategyType: strategyType,
		Underlying:   underlying,
	}

	for name, snap := range selectedLegs {
		sel, ok := selectors[name]
		if !ok {
			return nil, fmt.Errorf("leg_selector: missing selector for leg %q", name)
		}
		ratio := sel.Ratio
		if ratio < 1 {
			ratio = 1
		}
		leg := domain.SpreadLeg{
			Contract:       snap.Contract,
			Side:           sel.Side,
			PositionIntent: sel.Intent,
			Ratio:          ratio,
			Quantity:       float64(ratio),
		}
		spread.Legs = append(spread.Legs, leg)
	}

	return spread, nil
}

// daysToExpiry returns the number of calendar days from now until expiry.
func daysToExpiry(expiry, now time.Time) int {
	d := expiry.Sub(now).Hours() / 24
	if d < 0 {
		return 0
	}
	return int(d)
}
