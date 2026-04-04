package analysts

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// MarketAnalyst is an analysis-phase Node that calls the LLM with technical
// market data (OHLCV bars and indicators) and stores the resulting report in
// the pipeline state.
type MarketAnalyst struct{ BaseAnalyst }

// NewMarketAnalyst returns a MarketAnalyst wired to the given LLM provider and
// model. providerName (e.g. "openai") is recorded in decision metadata. A nil
// logger is replaced with the default logger.
func NewMarketAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *MarketAnalyst {
	return &MarketAnalyst{BaseAnalyst: NewBaseAnalyst(BaseAnalystConfig{
		Provider:     provider,
		ProviderName: providerName,
		Model:        model,
		Logger:       logger,
		Role:         agent.AgentRoleMarketAnalyst,
		Name:         "market_analyst",
		SystemPrompt: MarketAnalystSystemPrompt,
		BuildPrompt: func(input agent.AnalysisInput) (string, bool) {
			var bars []domain.OHLCV
			var indicators []domain.Indicator
			if input.Market != nil {
				bars = input.Market.Bars
				indicators = input.Market.Indicators
			}
			return FormatMarketAnalystUserPrompt(input.Ticker, bars, indicators), true
		},
	})}
}
