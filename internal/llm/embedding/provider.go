// Package embedding provides a Provider interface and implementations for
// generating vector embeddings from text. These embeddings are stored alongside
// triaged content (news, social sentiment) and used for semantic retrieval
// during strategy runs.
package embedding

import "context"

// Provider generates dense vector embeddings from text.
type Provider interface {
	// Embed returns a single embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch returns embedding vectors for each input text.
	// The returned slice has the same length and order as texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
