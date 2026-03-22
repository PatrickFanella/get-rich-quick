package binance

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func init() {
	data.RegisterBinanceProviderFactory(func(logger *slog.Logger) data.DataProvider {
		return NewProvider(logger)
	})
}
