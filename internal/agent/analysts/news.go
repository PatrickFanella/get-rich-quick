package analysts

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// NewsAnalyst is an analysis-phase Node that calls the LLM with recent news
// articles and stores the resulting sentiment and catalyst report in the
// pipeline state. When no news articles are available it skips the LLM call
// and stores a static message instead.
type NewsAnalyst struct{ BaseAnalyst }

// NewNewsAnalyst returns a NewsAnalyst wired to the given LLM provider and
// model. providerName (e.g. "openai") is recorded in decision metadata. A nil
// logger is replaced with the default logger.
func NewNewsAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *NewsAnalyst {
	return &NewsAnalyst{BaseAnalyst: NewBaseAnalyst(BaseAnalystConfig{
		Provider:     provider,
		ProviderName: providerName,
		Model:        model,
		Logger:       logger,
		Role:         agent.AgentRoleNewsAnalyst,
		Name:         "news_analyst",
		SystemPrompt: NewsAnalystSystemPrompt,
		SkipMessage:  "No news articles available. Unable to perform news analysis.",
		BuildPrompt: func(input agent.AnalysisInput) (string, bool) {
			if len(input.News) == 0 {
				return "", false
			}
			prompt := FormatNewsAnalystUserPrompt(input.Ticker, input.News)
			if input.PredictionMarket != nil {
				prompt += PolymarketNewsAnalystNote
			}
			return prompt, true
		},
	})}
}
