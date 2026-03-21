package data

import (
	"context"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

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

	// GetSocialSentiment returns aggregated social-media sentiment for a ticker.
	GetSocialSentiment(ctx context.Context, ticker string) (SocialSentiment, error)
}
