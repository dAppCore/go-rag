package rag

import "context"

// Embedder defines the interface for generating text embeddings.
// OllamaClient satisfies this interface.
type Embedder interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates embedding vectors for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedDimension returns the dimensionality of the embedding vectors
	// produced by the configured model.
	EmbedDimension() uint64
}
