package rag

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"dappco.re/go/core/log"
	"github.com/ollama/ollama/api"
)

// OllamaConfig holds Ollama connection configuration.
type OllamaConfig struct {
	Scheme string
	Host   string
	Port   int
	Model  string
}

// DefaultOllamaConfig returns default Ollama configuration.
// Host defaults to localhost for local development.
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		Scheme: "http",
		Host:   "localhost",
		Port:   11434,
		Model:  "nomic-embed-text",
	}
}

// OllamaClient wraps the Ollama API client for embeddings.
type OllamaClient struct {
	client *api.Client
	config OllamaConfig
}

// NewOllamaClient creates a new Ollama client.
func NewOllamaClient(cfg OllamaConfig) (*OllamaClient, error) {
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}
	baseURL := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	client := api.NewClient(baseURL, &http.Client{
		Timeout: 30 * time.Second,
	})

	return &OllamaClient{
		client: client,
		config: cfg,
	}, nil
}

// NewOllamaEmbedder creates an Ollama embedder from a base URL and model name.
func NewOllamaEmbedder(baseURL, model string) (*OllamaClient, error) {
	cfg, err := ollamaConfigFromEndpoint(baseURL)
	if err != nil {
		return nil, err
	}
	if model != "" {
		cfg.Model = model
	}
	return NewOllamaClient(cfg)
}

func ollamaConfigFromEndpoint(endpoint string) (OllamaConfig, error) {
	cfg := DefaultOllamaConfig()
	if endpoint == "" {
		return cfg, nil
	}

	parsed, err := parseEndpointURL(endpoint)
	if err != nil {
		return OllamaConfig{}, err
	}

	host := parsed.Hostname()
	if host == "" {
		host = cfg.Host
	}

	port := cfg.Port
	if p := parsed.Port(); p != "" {
		if parsedPort, err := strconv.Atoi(p); err == nil {
			port = parsedPort
		}
	}

	cfg.Scheme = parsed.Scheme
	if cfg.Scheme == "" {
		cfg.Scheme = "http"
	}
	cfg.Host = host
	cfg.Port = port
	return cfg, nil
}

// EmbedDimension returns the embedding dimension for the configured model.
// nomic-embed-text uses 768 dimensions.
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
func (o *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbedRequest{
		Model: o.config.Model,
		Input: text,
	}

	resp, err := o.client.Embed(ctx, req)
	if err != nil {
		return nil, log.E("rag.Ollama.Embed", "failed to generate embedding", err)
	}

	if len(resp.Embeddings) == 0 || len(resp.Embeddings[0]) == 0 {
		return nil, log.E("rag.Ollama.Embed", "empty embedding response", nil)
	}

	// Convert float64 to float32 for Qdrant
	embedding := resp.Embeddings[0]
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(v)
	}

	return result, nil
}

// EmbedBatch generates embeddings for multiple texts.
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
		return nil, log.E("rag.Ollama.EmbedBatch", "failed to generate batch embeddings", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, log.E("rag.Ollama.EmbedBatch", "empty embedding response", nil)
	}
	if len(resp.Embeddings) != len(texts) {
		return nil, log.E("rag.Ollama.EmbedBatch", fmt.Sprintf("unexpected embedding count: got %d, want %d", len(resp.Embeddings), len(texts)), nil)
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
func (o *OllamaClient) VerifyModel(ctx context.Context) error {
	_, err := o.Embed(ctx, "test")
	if err != nil {
		return log.E("rag.Ollama.VerifyModel", fmt.Sprintf("model %s not available (run: ollama pull %s)", o.config.Model, o.config.Model), err)
	}
	return nil
}

// Model returns the configured embedding model name.
func (o *OllamaClient) Model() string {
	return o.config.Model
}
