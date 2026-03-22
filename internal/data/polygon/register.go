package polygon

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func init() {
	data.RegisterPolygonProviderFactory(func(apiKey string, logger *slog.Logger) data.DataProvider {
		return NewProvider(NewClient(apiKey, logger))
	})
}
