package analysts

import (
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// SocialMediaAnalyst is an analysis-phase Node that calls the LLM with
// social-media sentiment data and stores the resulting report in the pipeline
// state.
type SocialMediaAnalyst struct{ BaseAnalyst }

// NewSocialMediaAnalyst returns a SocialMediaAnalyst wired to the given LLM
// provider and model. providerName (e.g. "openai") is recorded in decision
// metadata. A nil logger is replaced with the default logger.
func NewSocialMediaAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *SocialMediaAnalyst {
	return &SocialMediaAnalyst{BaseAnalyst: NewBaseAnalyst(BaseAnalystConfig{
		Provider:     provider,
		ProviderName: providerName,
		Model:        model,
		Logger:       logger,
		Role:         agent.AgentRoleSocialMediaAnalyst,
		Name:         "social_media_analyst",
		SystemPrompt: SocialAnalystSystemPrompt,
		SkipMessage:  "Social sentiment data unavailable for this ticker. Analysis skipped to conserve resources.",
		BuildPrompt: func(input agent.AnalysisInput) (string, bool) {
			return FormatSocialAnalystUserPrompt(input.Ticker, input.Social), input.Social != nil
		},
	})}
}
