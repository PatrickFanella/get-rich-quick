package stocktwits

import "github.com/PatrickFanella/get-rich-quick/internal/data"

// Register adds the StockTwits provider factory to the given registry.
// StockTwits is a social-sentiment-only provider using the public API.
// No API key is required.
func Register(reg *data.ProviderRegistry) {
	reg.StockTwits = func(cfg data.ProviderConfig) data.DataProvider {
		return NewDataProvider(cfg.Logger)
	}
}
