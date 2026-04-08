package polymarket

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

// Register adds the Polymarket data provider factory to the given registry.
func Register(reg *data.ProviderRegistry) {
	reg.Polymarket = func(clobURL string, logger *slog.Logger) data.DataProvider {
		return NewProvider(clobURL, logger)
	}
}
