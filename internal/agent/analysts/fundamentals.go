package analysts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

// FundamentalsAnalyst is an analysis-phase Node that calls the LLM with
// fundamental financial data and stores the resulting report in the pipeline
// state. When no fundamentals are available (e.g. for crypto assets) it stores
// a short explanatory message without making an LLM call.
type FundamentalsAnalyst struct {
	BaseAnalyst
}

// NewFundamentalsAnalyst returns a FundamentalsAnalyst wired to the given LLM
// provider and model. providerName (e.g. "openai") is recorded in decision
// metadata. A nil logger is replaced with the default logger.
func NewFundamentalsAnalyst(provider llm.Provider, providerName, model string, logger *slog.Logger) *FundamentalsAnalyst {
	return &FundamentalsAnalyst{
		BaseAnalyst: NewBaseAnalyst(provider, providerName, model, logger),
	}
}

// Name returns the human-readable name for this node.
func (f *FundamentalsAnalyst) Name() string { return "fundamentals_analyst" }

// Role returns the agent role used to key reports and decisions in the state.
func (f *FundamentalsAnalyst) Role() agent.AgentRole { return agent.AgentRoleFundamentalsAnalyst }

// Phase returns the pipeline phase this node belongs to.
func (f *FundamentalsAnalyst) Phase() agent.Phase { return agent.PhaseAnalysis }

// Execute formats the fundamentals data from state into a prompt, calls the
// LLM, and stores the analysis report in state. When state.Fundamentals is nil
// the node skips the LLM call and records a static message indicating that
// fundamental data is not available for this asset type.
func (f *FundamentalsAnalyst) Execute(ctx context.Context, state *agent.PipelineState) error {
	if f.provider == nil {
		return fmt.Errorf("fundamentals_analyst: provider is nil")
	}

	// When no fundamentals are available (e.g. crypto), skip the LLM call.
	if state.Fundamentals == nil {
		const msg = "No fundamentals available for this asset type."
		f.logger.InfoContext(ctx, "fundamentals analyst skipped: no fundamentals data")
		state.SetAnalystReport(f.Role(), msg)
		state.RecordDecision(f.Role(), f.Phase(), nil, msg, nil)
		return nil
	}

	userPrompt := FormatFundamentalsAnalystUserPrompt(
		state.Ticker,
		state.Fundamentals,
	)

	resp, err := f.provider.Complete(ctx, llm.CompletionRequest{
		Model: f.model,
		Messages: []llm.Message{
			{Role: "system", Content: FundamentalsAnalystSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return fmt.Errorf("fundamentals_analyst: llm completion failed: %w", err)
	}

	f.logger.InfoContext(ctx, "fundamentals analyst report generated",
		slog.Int("prompt_tokens", resp.Usage.PromptTokens),
		slog.Int("completion_tokens", resp.Usage.CompletionTokens),
	)

	state.SetAnalystReport(f.Role(), resp.Content)
	state.RecordDecision(f.Role(), f.Phase(), nil, resp.Content, &agent.DecisionLLMResponse{
		Provider: f.providerName,
		Response: resp,
	})

	return nil
}
