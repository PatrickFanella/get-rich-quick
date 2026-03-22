package yahoo

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func init() {
	data.RegisterYahooProviderFactory(func(logger *slog.Logger) data.DataProvider {
		return NewProvider(logger)
	})
}
