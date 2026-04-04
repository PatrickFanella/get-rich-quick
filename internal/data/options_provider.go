package data

import (
	"context"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// OptionsDataProvider retrieves options-specific market data.
type OptionsDataProvider interface {
	// GetOptionsChain returns option snapshots (price + Greeks) for an underlying.
	// If expiry is zero, all expiries are returned. If optionType is empty, both
	// calls and puts are included.
	GetOptionsChain(ctx context.Context, underlying string, expiry time.Time, optionType domain.OptionType) ([]domain.OptionSnapshot, error)

	// GetOptionsOHLCV returns historical OHLCV bars for a specific options contract.
	GetOptionsOHLCV(ctx context.Context, occSymbol string, timeframe Timeframe, from, to time.Time) ([]domain.OHLCV, error)
}
