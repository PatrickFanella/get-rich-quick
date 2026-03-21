package llm_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

type stubProvider struct{}

func (stubProvider) Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return nil, nil
}

func TestModelTierString(t *testing.T) {
	tests := []struct {
		tier llm.ModelTier
		want string
	}{
		{llm.ModelTierDeepThink, "deep_think"},
		{llm.ModelTierQuickThink, "quick_think"},
	}

	for _, tc := range tests {
		if got := tc.tier.String(); got != tc.want {
			t.Errorf("ModelTier(%q).String() = %q, want %q", tc.tier, got, tc.want)
		}
	}
}

func TestRegistryRegisterAndResolve(t *testing.T) {
	registry := llm.NewRegistry()
	provider := stubProvider{}

	err := registry.Register(" OpenAI ", provider, map[llm.ModelTier]string{
		llm.ModelTierDeepThink:  "gpt-5.2",
		llm.ModelTierQuickThink: "gpt-5-mini",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	entry, ok := registry.Get("openai")
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}
	if _, ok := entry.Provider.(stubProvider); !ok {
		t.Fatalf("Get() provider type = %T, want stubProvider", entry.Provider)
	}
	if got := entry.Models[llm.ModelTierDeepThink]; got != "gpt-5.2" {
		t.Errorf("Get() deep model = %q, want %q", got, "gpt-5.2")
	}

	resolvedProvider, model, err := registry.Resolve("OPENAI", llm.ModelTierQuickThink)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if _, ok := resolvedProvider.(stubProvider); !ok {
		t.Fatalf("Resolve() provider type = %T, want stubProvider", resolvedProvider)
	}
	if model != "gpt-5-mini" {
		t.Errorf("Resolve() model = %q, want %q", model, "gpt-5-mini")
	}
}

func TestRegistryResolveErrors(t *testing.T) {
	registry := llm.NewRegistry()

	_, _, err := registry.Resolve("missing", llm.ModelTierQuickThink)
	if !errors.Is(err, llm.ErrProviderNotFound) {
		t.Fatalf("Resolve() error = %v, want ErrProviderNotFound", err)
	}

	err = registry.Register("anthropic", stubProvider{}, map[llm.ModelTier]string{
		llm.ModelTierDeepThink: "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, _, err = registry.Resolve("anthropic", llm.ModelTierQuickThink)
	if !errors.Is(err, llm.ErrModelTierNotConfigured) {
		t.Fatalf("Resolve() error = %v, want ErrModelTierNotConfigured", err)
	}
}

func TestRegistryGetReturnsCopyOfModels(t *testing.T) {
	registry := llm.NewRegistry()

	err := registry.Register("google", stubProvider{}, map[llm.ModelTier]string{
		llm.ModelTierQuickThink: "gemini-2.5-flash",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	entry, ok := registry.Get("google")
	if !ok {
		t.Fatalf("Get() ok = false, want true")
	}

	entry.Models[llm.ModelTierQuickThink] = "mutated"

	_, model, err := registry.Resolve("google", llm.ModelTierQuickThink)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if model != "gemini-2.5-flash" {
		t.Errorf("Resolve() model after mutation = %q, want %q", model, "gemini-2.5-flash")
	}
}

func TestRegistryRegisterRejectsBlankOrMissingModels(t *testing.T) {
	tests := []struct {
		name   string
		models map[llm.ModelTier]string
		want   string
	}{
		{
			name:   "missing models",
			models: nil,
			want:   "llm models are required",
		},
		{
			name: "blank model",
			models: map[llm.ModelTier]string{
				llm.ModelTierQuickThink: "   ",
			},
			want: "llm model name is required for tier quick_think",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			registry := llm.NewRegistry()

			err := registry.Register("openai", stubProvider{}, tc.models)
			if err == nil {
				t.Fatal("Register() error = nil, want non-nil")
			}
			if err.Error() != tc.want {
				t.Fatalf("Register() error = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestCompletionRequestJSONOmitsResponseFormatWhenUnset(t *testing.T) {
	payload, err := json.Marshal(llm.CompletionRequest{
		Model: "gpt-5-mini",
		Messages: []llm.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	if strings.Contains(string(payload), "response_format") {
		t.Fatalf("json payload = %s, want response_format omitted", payload)
	}
}

func TestCompletionResponseJSONUsesLatencyMS(t *testing.T) {
	payload, err := json.Marshal(llm.CompletionResponse{
		Content:   "done",
		Usage:     llm.CompletionUsage{PromptTokens: 10, CompletionTokens: 5},
		Model:     "gpt-5-mini",
		LatencyMS: 1234,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	serialized := string(payload)
	if !strings.Contains(serialized, `"latency_ms":1234`) {
		t.Fatalf("json payload = %s, want latency_ms field", payload)
	}
	if strings.Contains(serialized, `"latency":`) {
		t.Fatalf("json payload = %s, want latency field omitted", payload)
	}
}
