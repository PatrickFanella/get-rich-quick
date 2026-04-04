package agent

import (
	"fmt"
	"math"
)

// ValidateConfidence01 returns an error if v is outside the [0, 1] range or NaN.
func ValidateConfidence01(v float64) error {
	if math.IsNaN(v) || v < 0 || v > 1 {
		return fmt.Errorf("confidence must be in [0, 1], got %v", v)
	}
	return nil
}
