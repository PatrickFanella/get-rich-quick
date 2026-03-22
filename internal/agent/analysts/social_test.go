package analysts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/data"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

func TestSocialMediaAnalystName(t *testing.T) {
	sa := NewSocialMediaAnalyst(&mockProvider{}, "openai", "test-model", nil)
	if sa.Name() != "social_media_analyst" {
		t.Errorf("Name() = %q, want %q", sa.Name(), "social_media_analyst")
	}
}

func TestSocialMediaAnalystRole(t *testing.T) {
	sa := NewSocialMediaAnalyst(&mockProvider{}, "openai", "test-model", nil)
	if sa.Role() != agent.AgentRoleSocialMediaAnalyst {
		t.Errorf("Role() = %q, want %q", sa.Role(), agent.AgentRoleSocialMediaAnalyst)
	}
}

func TestSocialMediaAnalystPhase(t *testing.T) {
	sa := NewSocialMediaAnalyst(&mockProvider{}, "openai", "test-model", nil)
	if sa.Phase() != agent.PhaseAnalysis {
		t.Errorf("Phase() = %q, want %q", sa.Phase(), agent.PhaseAnalysis)
	}
}

func TestSocialMediaAnalystExecute(t *testing.T) {
	wantContent := "## Retail Sentiment Summary\nBullish retail sentiment detected."
	mock := &mockProvider{
		response: &llm.CompletionResponse{
			Content: wantContent,
			Usage: llm.CompletionUsage{
				PromptTokens:     90,
				CompletionTokens: 60,
			},
			Model: "gpt-4",
		},
	}

	sa := NewSocialMediaAnalyst(mock, "openai", "gpt-4", nil)

	state := &agent.PipelineState{
		Ticker: "AAPL",
		Social: &data.SocialSentiment{
			Ticker:       "AAPL",
			Score:        0.75,
			Bullish:      0.65,
			Bearish:      0.20,
			PostCount:    1200,
			CommentCount: 4500,
			MeasuredAt:   time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC),
		},
	}

	err := sa.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	// Verify LLM was called exactly once.
	if got := mock.calls.Load(); got != 1 {
		t.Errorf("LLM called %d times, want 1", got)
	}

	// Verify the request contains system and user messages.
	if len(mock.lastReq.Messages) != 2 {
		t.Fatalf("request has %d messages, want 2", len(mock.lastReq.Messages))
	}
	if mock.lastReq.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", mock.lastReq.Messages[0].Role, "system")
	}
	if mock.lastReq.Messages[0].Content != SocialAnalystSystemPrompt {
		t.Error("system message does not match SocialAnalystSystemPrompt")
	}
	if mock.lastReq.Messages[1].Role != "user" {
		t.Errorf("second message role = %q, want %q", mock.lastReq.Messages[1].Role, "user")
	}

	// Verify user prompt references the ticker and social data.
	userMsg := mock.lastReq.Messages[1].Content
	for _, want := range []string{"AAPL", "0.7500", "0.6500", "1200", "4500"} {
		if !strings.Contains(userMsg, want) {
			t.Errorf("user prompt missing expected content %q", want)
		}
	}

	// Verify model was passed through.
	if mock.lastReq.Model != "gpt-4" {
		t.Errorf("request model = %q, want %q", mock.lastReq.Model, "gpt-4")
	}

	// Verify report stored in state.
	report, ok := state.AnalystReports[agent.AgentRoleSocialMediaAnalyst]
	if !ok {
		t.Fatal("analyst report not stored in state")
	}
	if report != wantContent {
		t.Errorf("stored report = %q, want %q", report, wantContent)
	}

	// Verify decision recorded in state.
	dec, ok := state.Decision(agent.AgentRoleSocialMediaAnalyst, agent.PhaseAnalysis, nil)
	if !ok {
		t.Fatal("decision not recorded in state")
	}
	if dec.OutputText != wantContent {
		t.Errorf("decision output = %q, want %q", dec.OutputText, wantContent)
	}
	if dec.LLMResponse == nil {
		t.Fatal("decision LLM response is nil")
	}
	if dec.LLMResponse.Response.Usage.PromptTokens != 90 {
		t.Errorf("prompt tokens = %d, want 90", dec.LLMResponse.Response.Usage.PromptTokens)
	}
	if dec.LLMResponse.Provider != "openai" {
		t.Errorf("decision provider = %q, want %q", dec.LLMResponse.Provider, "openai")
	}
}

func TestSocialMediaAnalystExecuteNilSocialData(t *testing.T) {
	wantContent := "No social sentiment data available for analysis."
	mock := &mockProvider{
		response: &llm.CompletionResponse{
			Content: wantContent,
			Usage: llm.CompletionUsage{
				PromptTokens:     50,
				CompletionTokens: 30,
			},
			Model: "gpt-4",
		},
	}

	sa := NewSocialMediaAnalyst(mock, "openai", "gpt-4", nil)
	state := &agent.PipelineState{Ticker: "BTC"}

	err := sa.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	// Verify LLM was still called.
	if got := mock.calls.Load(); got != 1 {
		t.Errorf("LLM called %d times, want 1", got)
	}

	// Verify the user prompt indicates missing data.
	userMsg := mock.lastReq.Messages[1].Content
	if !strings.Contains(userMsg, "No social sentiment data available") {
		t.Errorf("user prompt should indicate missing social data, got: %s", userMsg)
	}

	// Verify report stored in state.
	report, ok := state.AnalystReports[agent.AgentRoleSocialMediaAnalyst]
	if !ok {
		t.Fatal("analyst report not stored in state")
	}
	if report != wantContent {
		t.Errorf("stored report = %q, want %q", report, wantContent)
	}
}

func TestSocialMediaAnalystExecuteLLMError(t *testing.T) {
	mock := &mockProvider{
		err: errors.New("rate limit exceeded"),
	}

	sa := NewSocialMediaAnalyst(mock, "openai", "gpt-4", nil)
	state := &agent.PipelineState{Ticker: "ETH"}

	err := sa.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() should return error when LLM fails")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("error should wrap LLM error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "social_media_analyst") {
		t.Errorf("error should include node name, got: %v", err)
	}
}

func TestSocialMediaAnalystExecuteNilProvider(t *testing.T) {
	sa := NewSocialMediaAnalyst(nil, "openai", "gpt-4", nil)
	state := &agent.PipelineState{Ticker: "GOOG"}

	err := sa.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() should return error when provider is nil")
	}
	if !strings.Contains(err.Error(), "provider is nil") {
		t.Errorf("error should mention nil provider, got: %v", err)
	}
}

func TestSocialMediaAnalystImplementsNode(t *testing.T) {
	// Compile-time check that *SocialMediaAnalyst satisfies agent.Node.
	var _ agent.Node = (*SocialMediaAnalyst)(nil)
}
