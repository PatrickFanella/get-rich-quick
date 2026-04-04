package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseOCC parses a standard OCC option symbol into an OptionContract.
// Format: {ROOT up to 6}{YYMMDD}{C|P}{strike*1000, 8 digits}
// Example: AAPL241220C00150000 → AAPL, 2024-12-20, call, $150.00
// Also accepts the Massive/Polygon "O:" prefix: O:AAPL241220C00150000
func ParseOCC(symbol string) (*OptionContract, error) {
	sym := strings.TrimPrefix(symbol, "O:")
	if len(sym) < 15 {
		return nil, fmt.Errorf("occ: symbol %q too short (min 15 chars)", symbol)
	}

	// The last 9 characters are: C/P (1) + strike (8)
	// Before that are 6 date chars: YYMMDD
	// Everything before the date is the root symbol (1-6 chars)
	strikeStr := sym[len(sym)-8:]
	cpChar := sym[len(sym)-9 : len(sym)-8]
	dateStr := sym[len(sym)-15 : len(sym)-9]
	root := sym[:len(sym)-15]

	if root == "" || len(root) > 6 {
		return nil, fmt.Errorf("occ: invalid root symbol %q in %q", root, symbol)
	}

	var optType OptionType
	switch cpChar {
	case "C":
		optType = OptionTypeCall
	case "P":
		optType = OptionTypePut
	default:
		return nil, fmt.Errorf("occ: invalid option type %q in %q (expected C or P)", cpChar, symbol)
	}

	year, err := strconv.Atoi(dateStr[0:2])
	if err != nil {
		return nil, fmt.Errorf("occ: invalid year in %q: %w", symbol, err)
	}
	month, err := strconv.Atoi(dateStr[2:4])
	if err != nil {
		return nil, fmt.Errorf("occ: invalid month in %q: %w", symbol, err)
	}
	day, err := strconv.Atoi(dateStr[4:6])
	if err != nil {
		return nil, fmt.Errorf("occ: invalid day in %q: %w", symbol, err)
	}
	expiry := time.Date(2000+year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	strikeInt, err := strconv.ParseInt(strikeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("occ: invalid strike in %q: %w", symbol, err)
	}
	strike := float64(strikeInt) / 1000.0

	return &OptionContract{
		OCCSymbol:  sym,
		Underlying: root,
		OptionType: optType,
		Strike:     strike,
		Expiry:     expiry,
		Multiplier: 100,
		Style:      "american",
	}, nil
}

// FormatOCC builds a standard OCC symbol from components.
func FormatOCC(underlying string, optType OptionType, strike float64, expiry time.Time) string {
	cp := "C"
	if optType == OptionTypePut {
		cp = "P"
	}
	strikeInt := int64(strike * 1000)
	return fmt.Sprintf("%s%s%s%08d",
		strings.ToUpper(underlying),
		expiry.Format("060102"),
		cp,
		strikeInt,
	)
}

// MassiveSymbol prepends the "O:" prefix used by Polygon/Massive API.
func MassiveSymbol(occ string) string {
	if strings.HasPrefix(occ, "O:") {
		return occ
	}
	return "O:" + occ
}

// AlpacaSymbol strips the "O:" prefix if present, returning raw OCC for Alpaca.
func AlpacaSymbol(occ string) string {
	return strings.TrimPrefix(occ, "O:")
}
