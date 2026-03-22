package analysts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// SocialMediaAnalyst is an analysis-phase Node that calls the LLM with
// social-media sentiment data and stores the resulting report in the pipeline
// state.
type SocialMediaAnalyst struct {
	BaseAnalyst
}

// NewSocialMediaAnalyst returns a SocialMediaAnalyst wired to the given LLM
// provider and model. providerName (e.g. "openai") is recorded in decision
// metadata. A nil logger is replaced with the default logger.
func NewSocialMediaAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *SocialMediaAnalyst {
	return &SocialMediaAnalyst{
		BaseAnalyst: NewBaseAnalyst(provider, providerName, model, logger),
	}
}

// Name returns the human-readable name for this node.
func (s *SocialMediaAnalyst) Name() string { return "social_media_analyst" }

// Role returns the agent role used to key reports and decisions in the state.
func (s *SocialMediaAnalyst) Role() agent.AgentRole { return agent.AgentRoleSocialMediaAnalyst }

// Phase returns the pipeline phase this node belongs to.
func (s *SocialMediaAnalyst) Phase() agent.Phase { return agent.PhaseAnalysis }

// Execute formats the social sentiment data from state into a prompt, calls
// the LLM, and stores the analysis report in state. When social data is nil
// the analyst still calls the LLM so it can produce a report noting the
// absence of data.
func (s *SocialMediaAnalyst) Execute(ctx context.Context, state *agent.PipelineState) error {
	if s.provider == nil {
		return fmt.Errorf("social_media_analyst: provider is nil")
	}

	userPrompt := FormatSocialAnalystUserPrompt(state.Ticker, state.Social)

	resp, err := s.provider.Complete(ctx, llm.CompletionRequest{
		Model: s.model,
		Messages: []llm.Message{
			{Role: "system", Content: SocialAnalystSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return fmt.Errorf("social_media_analyst: llm completion failed: %w", err)
	}

	s.logger.InfoContext(ctx, "social media analyst report generated",
		slog.Int("prompt_tokens", resp.Usage.PromptTokens),
		slog.Int("completion_tokens", resp.Usage.CompletionTokens),
	)

	state.SetAnalystReport(s.Role(), resp.Content)
	state.RecordDecision(s.Role(), s.Phase(), nil, resp.Content, &agent.DecisionLLMResponse{
		Provider: s.providerName,
		Response: resp,
	})

	return nil
}
