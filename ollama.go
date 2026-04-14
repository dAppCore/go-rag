package rag

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"dappco.re/go/core"
	"github.com/ollama/ollama/api"
)

// OllamaConfig holds Ollama connection configuration.
// cfg := OllamaConfig{Host: "localhost", Port: 11434, Model: "nomic-embed-text"}
type OllamaConfig struct {
	Host  string
	Port  int
	Model string
}

// DefaultOllamaConfig returns default Ollama configuration.
// Host defaults to localhost for local development.
// cfg := DefaultOllamaConfig()
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		Host:  "localhost",
		Port:  11434,
		Model: "nomic-embed-text",
	}
}

// OllamaClient wraps the Ollama API client for embeddings.
// client, _ := NewOllamaClient(DefaultOllamaConfig())
type OllamaClient struct {
	client *api.Client
	config OllamaConfig
}

// NewOllamaClient creates a new Ollama client.
// client, err := NewOllamaClient(DefaultOllamaConfig())
func NewOllamaClient(cfg OllamaConfig) (*OllamaClient, error) {
	baseURL := &url.URL{
		Scheme: "http",
		Host:   core.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	client := api.NewClient(baseURL, &http.Client{
		Timeout: 30 * time.Second,
	})

	return &OllamaClient{
		client: client,
		config: cfg,
	}, nil
}

// EmbedDimension returns the embedding dimension for the configured model.
// nomic-embed-text uses 768 dimensions.
// dim := client.EmbedDimension()
func (o *OllamaClient) EmbedDimension() uint64 {
	switch o.config.Model {
	case "nomic-embed-text":
		return 768
	case "mxbai-embed-large":
		return 1024
	case "all-minilm":
		return 384
	default:
		return 768 // Default to nomic-embed-text dimension
	}
}

// Embed generates embeddings for the given text.
// vector, _ := client.Embed(ctx, "How do goroutines work?")
func (o *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbedRequest{
		Model: o.config.Model,
		Input: text,
	}

	resp, err := o.client.Embed(ctx, req)
	if err != nil {
		return nil, core.E("rag.Ollama.Embed", "failed to generate embedding", err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0]) == 0 {
		return nil, core.E("rag.Ollama.Embed", "empty embedding response", nil)
	}

	// Convert float64 to float32 for Qdrant
	embedding := resp.Embeddings[0]
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

// EmbedBatch generates embeddings for multiple texts in a single Ollama
// request instead of looping per text — avoids N round-trips for large batches.
//
// vectors, _ := client.EmbedBatch(ctx, []string{"How do goroutines work?", "What is Qdrant?"})
func (o *OllamaClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	req := &api.EmbedRequest{
		Model: o.config.Model,
		Input: texts,
	}

	resp, err := o.client.Embed(ctx, req)
	if err != nil {
		return nil, core.E("rag.Ollama.EmbedBatch", "failed to generate batch embeddings", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, core.E("rag.Ollama.EmbedBatch", "empty embedding response", nil)
	}
	if len(resp.Embeddings) != len(texts) {
		return nil, core.E("rag.Ollama.EmbedBatch", core.Sprintf("unexpected embedding count: got %d, want %d", len(resp.Embeddings), len(texts)), nil)
	}

	results := make([][]float32, len(resp.Embeddings))
	for i, embedding := range resp.Embeddings {
		vec := make([]float32, len(embedding))
		copy(vec, embedding)
		results[i] = vec
	}
	return results, nil
}

// VerifyModel checks if the embedding model is available.
// client.VerifyModel(ctx)
func (o *OllamaClient) VerifyModel(ctx context.Context) error {
	_, err := o.Embed(ctx, "test")
	if err != nil {
		return core.E(
			"rag.Ollama.VerifyModel",
			core.Sprintf("model %s not available (run: ollama pull %s)", o.config.Model, o.config.Model),
			err,
		)
	}
	return nil
}

// Model returns the configured embedding model name.
// name := client.Model()
func (o *OllamaClient) Model() string {
	return o.config.Model
}
