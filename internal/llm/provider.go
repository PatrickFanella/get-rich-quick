package llm

import "context"

// Provider defines the minimal interface required to execute an LLM completion.
type Provider interface {
	Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)
}

// ProviderFunc adapts a plain function to the Provider interface.
type ProviderFunc func(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)

// Complete delegates to the wrapped function.
func (f ProviderFunc) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	return f(ctx, request)
}
