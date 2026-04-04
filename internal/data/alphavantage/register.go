package alphavantage

import (
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

// Register adds the Alpha Vantage provider factory to the given registry.
func Register(reg *data.ProviderRegistry) {
	reg.AlphaVantage = func(apiKey string, rateLimitPerMinute int, logger *slog.Logger) data.DataProvider {
		if rateLimitPerMinute > 0 {
			return NewProvider(NewClient(apiKey, logger, data.NewRateLimiter(rateLimitPerMinute, time.Minute)))
		}

		return NewProvider(NewClient(apiKey, logger))
	}
}
