package data

import (
	"context"
	"errors"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// ErrNotImplemented indicates a provider does not support a requested method.
var ErrNotImplemented = errors.New("data: not implemented")

// DataProvider defines the abstraction for retrieving market data.
// A provider may support a subset of methods; unsupported methods should
// return a non-nil error so that ProviderChain can fall back to the next provider.
type DataProvider interface {
	// GetOHLCV returns candlestick bars for the given ticker and timeframe
	// between from and to (inclusive).
	GetOHLCV(ctx context.Context, ticker string, timeframe Timeframe, from, to time.Time) ([]domain.OHLCV, error)

	// GetFundamentals returns the latest fundamental data for a ticker.
	GetFundamentals(ctx context.Context, ticker string) (Fundamentals, error)

	// GetNews returns news articles for the given ticker between from and to.
	GetNews(ctx context.Context, ticker string, from, to time.Time) ([]NewsArticle, error)

	// GetSocialSentiment returns social-media sentiment snapshots for a ticker
	// between from and to (inclusive).
	GetSocialSentiment(ctx context.Context, ticker string, from, to time.Time) ([]SocialSentiment, error)
}
