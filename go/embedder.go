package rag

import (
	"context"

	"dappco.re/go"
)

// Embedder defines the interface for generating text embeddings.
// var embedder Embedder = ollamaClient
type Embedder interface {
	// Embed generates an embedding vector for the given text.
	Embed(ctx context.Context, text string) core.Result

	// EmbedBatch generates embedding vectors for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) core.Result

	// EmbedDimension returns the dimensionality of the embedding vectors
	// produced by the configured model.
	EmbedDimension() uint64
}
