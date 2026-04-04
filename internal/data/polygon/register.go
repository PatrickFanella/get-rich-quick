package polygon

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

// Register adds the Polygon provider factory to the given registry.
func Register(reg *data.ProviderRegistry) {
	reg.Polygon = func(apiKey string, logger *slog.Logger) data.DataProvider {
		return NewProvider(NewClient(apiKey, logger))
	}
}
