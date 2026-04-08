package analysts

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// FundamentalsAnalyst is an analysis-phase Node that calls the LLM with
// fundamental financial data and stores the resulting report in the pipeline
// state. When no fundamentals are available (e.g. for crypto assets) it stores
// a short explanatory message without making an LLM call.
type FundamentalsAnalyst struct{ BaseAnalyst }

// NewFundamentalsAnalyst returns a FundamentalsAnalyst wired to the given LLM
// provider and model. providerName (e.g. "openai") is recorded in decision
// metadata. A nil logger is replaced with the default logger.
func NewFundamentalsAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *FundamentalsAnalyst {
	return &FundamentalsAnalyst{BaseAnalyst: NewBaseAnalyst(BaseAnalystConfig{
		Provider:     provider,
		ProviderName: providerName,
		Model:        model,
		Logger:       logger,
		Role:         agent.AgentRoleFundamentalsAnalyst,
		Name:         "fundamentals_analyst",
		SystemPrompt: FundamentalsAnalystSystemPrompt,
		SkipMessage:  "No fundamentals available for this asset type.",
		BuildSystemPrompt: func(input agent.AnalysisInput) string {
			return polymarketSystemPromptFor(input.PredictionMarket != nil)
		},
		BuildPrompt: func(input agent.AnalysisInput) (string, bool) {
			if input.PredictionMarket != nil {
				return FormatPolymarketFundamentalsUserPrompt(input), true
			}
			if input.Fundamentals == nil {
				return "", false
			}
			return FormatFundamentalsAnalystUserPrompt(input.Ticker, input.Fundamentals), true
		},
	})}
}
