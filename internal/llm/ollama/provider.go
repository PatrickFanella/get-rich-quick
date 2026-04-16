package ollama

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/llm"
	"github.com/PatrickFanella/get-rich-quick/internal/llm/parse"
	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const (
	// DefaultBaseURL is the default Ollama server address including the /v1 path
	// prefix required by Ollama's OpenAI-compatible chat completions endpoint.
	DefaultBaseURL = "http://localhost:11434/v1"

	// ModelLlama3 is the default Llama 3 model served by Ollama.
	ModelLlama3 = "llama3.2"
	// ModelMistral is the Mistral model available on Ollama.
	ModelMistral = "mistral"

	// ollamaAPIKey is a placeholder key used to satisfy the OpenAI-compatible API
	// client, which requires a non-empty API key even though Ollama does not
	// authenticate requests.
	ollamaAPIKey = "ollama"
)

// Config contains the settings required to create an Ollama provider.
type Config struct {
	// BaseURL is the address of the locally running Ollama server.
	// Defaults to DefaultBaseURL if empty.
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// Provider implements llm.Provider using Ollama's OpenAI-compatible HTTP API.
type Provider struct {
	client openaisdk.Client
	model  string
}

var _ llm.Provider = (*Provider)(nil)

// DefaultModelsByTier returns the default Ollama model mapping for the registry.
func DefaultModelsByTier() map[llm.ModelTier]string {
	return map[llm.ModelTier]string{
		llm.ModelTierDeepThink:  ModelLlama3,
		llm.ModelTierQuickThink: ModelLlama3,
	}
}

// NewProvider constructs an Ollama completion provider.
func NewProvider(cfg Config) (*Provider, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	opts := []option.RequestOption{
		option.WithAPIKey(ollamaAPIKey),
		option.WithBaseURL(baseURL),
		option.WithMaxRetries(0),
	}
	if cfg.HTTPClient != nil {
		opts = append(opts, option.WithHTTPClient(cfg.HTTPClient))
	}

	return &Provider{
		client: openaisdk.NewClient(opts...),
		model:  strings.TrimSpace(cfg.Model),
	}, nil
}

// Complete sends a chat completion request to the local Ollama server and returns
// the first response choice.
func (p *Provider) Complete(ctx context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if p == nil {
		return nil, errors.New("ollama: provider is nil")
	}
	if len(request.Messages) == 0 {
		return nil, errors.New("ollama: at least one message is required")
	}

	model := strings.TrimSpace(request.Model)
	if model == "" {
		model = p.model
	}
	if model == "" {
		return nil, errors.New("ollama: model is required")
	}

	messages, err := toChatCompletionMessages(request.Messages)
	if err != nil {
		return nil, err
	}

	responseFormat, err := toResponseFormat(request.ResponseFormat)
	if err != nil {
		return nil, err
	}

	params := openaisdk.ChatCompletionNewParams{
		Model:    shared.ChatModel(model),
		Messages: messages,
	}
	if request.Temperature != 0 {
		params.Temperature = openaisdk.Float(request.Temperature)
	}
	if request.MaxTokens > 0 {
		params.MaxCompletionTokens = openaisdk.Int(int64(request.MaxTokens))
	}
	if responseFormat != nil {
		params.ResponseFormat = *responseFormat
	}

	startedAt := time.Now()
	completion, err := p.client.Chat.Completions.New(ctx, params,
		// Disable qwen3 thinking mode so the model returns content directly
		// instead of consuming all tokens on internal chain-of-thought reasoning.
		option.WithJSONSet("think", false),
	)
	if err != nil {
		return nil, fmt.Errorf("ollama: complete request: %w", err)
	}
	if len(completion.Choices) == 0 {
		return nil, errors.New("ollama: completion response did not include any choices")
	}

	content := extractContent(completion.Choices[0].Message)

	return &llm.CompletionResponse{
		Content: parse.StripThinkingTags(content),
		Usage: llm.CompletionUsage{
			PromptTokens:     int(completion.Usage.PromptTokens),
			CompletionTokens: int(completion.Usage.CompletionTokens),
		},
		Model:     completion.Model,
		LatencyMS: int(time.Since(startedAt).Milliseconds()),
	}, nil
}

// extractContent returns the usable text from a chat completion message.
// Ollama's OpenAI-compatible endpoint puts qwen3's chain-of-thought output
// into a non-standard "reasoning" JSON field and leaves "content" empty.
// When that happens we fall back to the reasoning field.
func extractContent(msg openaisdk.ChatCompletionMessage) string {
	if msg.Content != "" {
		return msg.Content
	}

	reasoningField, ok := msg.JSON.ExtraFields["reasoning"]
	if !ok || !reasoningField.Valid() {
		return ""
	}

	raw := reasoningField.Raw()
	var reasoning string
	if err := json.Unmarshal([]byte(raw), &reasoning); err != nil {
		slog.Warn("ollama: failed to unmarshal reasoning field", "raw", raw, "error", err)
		return ""
	}

	if reasoning != "" {
		slog.Info("ollama: using reasoning field as content (thinking mode workaround)")
	}
	return reasoning
}

func toChatCompletionMessages(messages []llm.Message) ([]openaisdk.ChatCompletionMessageParamUnion, error) {
	chatMessages := make([]openaisdk.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, message := range messages {
		switch role := strings.ToLower(strings.TrimSpace(message.Role)); role {
		case "system":
			chatMessages = append(chatMessages, openaisdk.SystemMessage(message.Content))
		case "user":
			chatMessages = append(chatMessages, openaisdk.UserMessage(message.Content))
		case "assistant":
			chatMessages = append(chatMessages, openaisdk.AssistantMessage(message.Content))
		default:
			return nil, fmt.Errorf("ollama: unsupported message role %q", message.Role)
		}
	}

	return chatMessages, nil
}

func toResponseFormat(format *llm.ResponseFormat) (*openaisdk.ChatCompletionNewParamsResponseFormatUnion, error) {
	if format == nil {
		return nil, nil
	}

	switch format.Type {
	case "", llm.ResponseFormatText:
		return nil, nil
	case llm.ResponseFormatJSONObject:
		jsonObject := shared.NewResponseFormatJSONObjectParam()
		return &openaisdk.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &jsonObject,
		}, nil
	default:
		return nil, fmt.Errorf("ollama: unsupported response format type %q", format.Type)
	}
}
