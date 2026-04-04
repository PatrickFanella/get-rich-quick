package yahoo

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

// Register adds the Yahoo provider factory to the given registry.
func Register(reg *data.ProviderRegistry) {
	reg.Yahoo = func(logger *slog.Logger) data.DataProvider {
		return NewProvider(logger)
	}
}
