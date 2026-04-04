package agent

import (
	"math"
	"testing"
)

func TestValidateConfidence01(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       float64
		wantErr bool
	}{
		{"zero", 0, false},
		{"mid", 0.5, false},
		{"one", 1, false},
		{"negative", -0.1, true},
		{"above_one", 1.1, true},
		{"NaN", math.NaN(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateConfidence01(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfidence01(%v) error = %v, wantErr %v", tt.v, err, tt.wantErr)
			}
		})
	}
}
