package binance

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

// Register adds the Binance provider factory to the given registry.
func Register(reg *data.ProviderRegistry) {
	reg.Binance = func(logger *slog.Logger) data.DataProvider {
		return NewProvider(logger)
	}
}
