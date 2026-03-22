package alphavantage

import (
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func init() {
	data.RegisterAlphaVantageProviderFactory(func(apiKey string, rateLimitPerMinute int, logger *slog.Logger) data.DataProvider {
		if rateLimitPerMinute > 0 {
			return NewProvider(NewClient(apiKey, logger, data.NewRateLimiter(rateLimitPerMinute, time.Minute)))
		}

		return NewProvider(NewClient(apiKey, logger))
	})
}
