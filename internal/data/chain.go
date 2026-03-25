package data

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// ErrNoProviders is returned when a ProviderChain contains no providers.
var ErrNoProviders = errors.New("data: no providers in chain")

// ProviderChain tries each DataProvider in order and returns the first
// successful result. If all providers fail, the last error is returned.
type ProviderChain struct {
	providers []DataProvider
	logger    *slog.Logger
}

// NewProviderChain constructs a ProviderChain from an ordered list of providers.
// Providers are tried in the order they are given. If logger is nil, slog.Default() is used.
func NewProviderChain(logger *slog.Logger, providers ...DataProvider) *ProviderChain {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProviderChain{
		providers: providers,
		logger:    logger,
	}
}

// tryChain iterates providers calling fn for each one. It returns the result of
// the first successful call, or the last error if all providers fail.
func tryChain[T any](c *ProviderChain, method, ticker string, fn func(DataProvider) (T, error)) (T, error) {
	var zero T
	if len(c.providers) == 0 {
		return zero, ErrNoProviders
	}

	var lastErr error
	for _, p := range c.providers {
		result, err := fn(p)
		if err == nil {
			return result, nil
		}

		c.logger.Warn("data provider failed, trying next",
			slog.String("method", method),
			slog.String("ticker", ticker),
			slog.Any("error", err),
		)
		lastErr = err
	}

	return zero, lastErr
}

// GetOHLCV iterates providers and returns the first successful OHLCV result.
func (c *ProviderChain) GetOHLCV(ctx context.Context, ticker string, timeframe Timeframe, from, to time.Time) ([]domain.OHLCV, error) {
	return tryChain(c, "GetOHLCV", ticker, func(p DataProvider) ([]domain.OHLCV, error) {
		return p.GetOHLCV(ctx, ticker, timeframe, from, to)
	})
}

// GetFundamentals iterates providers and returns the first successful Fundamentals result.
func (c *ProviderChain) GetFundamentals(ctx context.Context, ticker string) (Fundamentals, error) {
	return tryChain(c, "GetFundamentals", ticker, func(p DataProvider) (Fundamentals, error) {
		return p.GetFundamentals(ctx, ticker)
	})
}

// GetNews iterates providers and returns the first successful news result.
func (c *ProviderChain) GetNews(ctx context.Context, ticker string, from, to time.Time) ([]NewsArticle, error) {
	return tryChain(c, "GetNews", ticker, func(p DataProvider) ([]NewsArticle, error) {
		return p.GetNews(ctx, ticker, from, to)
	})
}

// GetSocialSentiment iterates providers and returns the first successful sentiment result.
func (c *ProviderChain) GetSocialSentiment(ctx context.Context, ticker string, from, to time.Time) ([]SocialSentiment, error) {
	return tryChain(c, "GetSocialSentiment", ticker, func(p DataProvider) ([]SocialSentiment, error) {
		return p.GetSocialSentiment(ctx, ticker, from, to)
	})
}
