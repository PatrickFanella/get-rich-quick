package reddit

import "github.com/PatrickFanella/get-rich-quick/internal/data"

// Register adds the Reddit provider factory to the given registry.
// Reddit is a social-sentiment-only provider backed by public RSS feeds
// and LLM-based sentiment classification. No API key is required.
func Register(reg *data.ProviderRegistry) {
	reg.Reddit = func(cfg data.ProviderConfig) data.DataProvider {
		return NewProvider(cfg.LLMProvider, cfg.LLMModel, nil, cfg.Logger)
	}
}
