package rag

import (
	// Note: AX-6 - Ollama embedding calls propagate cancellation through context.Context.
	"context"
	// Note: AX-6 - Ollama's API constructor requires a standard *http.Client boundary.
	"net/http"
	// Note: AX-6 - Ollama's API constructor requires *url.URL; core has no URL primitive.
	"net/url"
	// Note: AX-6 - http.Client timeout is expressed as time.Duration; core has no duration primitive.
	"time"

	"dappco.re/go/core"
	"github.com/ollama/ollama/api"
)

// OllamaConfig holds Ollama connection configuration.
// cfg := OllamaConfig{Host: "localhost", Port: 11434, Model: "nomic-embed-text"}
type OllamaConfig struct {
	Scheme string
	Host   string
	Port   int
	Model  string
}

// DefaultOllamaConfig returns default Ollama configuration.
// Host defaults to localhost for local development.
// cfg := DefaultOllamaConfig()
func DefaultOllamaConfig() OllamaConfig {
	return OllamaConfig{
		Scheme: "http",
		Host:   "localhost",
		Port:   11434,
		Model:  "nomic-embed-text",
	}
}

// NewOllamaEmbedder creates an Ollama client from a base endpoint URL string.
// endpoint := "http://localhost:11434"
func NewOllamaEmbedder(endpoint string, model string) (*OllamaClient, error) {
	cfg, err := ollamaConfigFromEndpoint(endpoint)
	if err != nil {
		return nil, core.E("rag.NewOllamaEmbedder", "invalid Ollama endpoint", err)
	}
	if model != "" {
		cfg.Model = model
	}
	return NewOllamaClient(cfg)
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
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}
	baseURL := &url.URL{
		Scheme: scheme,
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

func ollamaConfigFromEndpoint(endpoint string) (OllamaConfig, error) {
	cfg := DefaultOllamaConfig()
	parsed, err := parseEndpointURL(endpoint)
	if err != nil {
		return OllamaConfig{}, err
	}

	host := parsed.Hostname()
	if host == "" {
		host = cfg.Host
	}
	cfg.Host = host

	if portText := parsed.Port(); portText != "" {
		port, err := parseEndpointPort("rag.ollamaConfigFromEndpoint", portText)
		if err != nil {
			return OllamaConfig{}, err
		}
		cfg.Port = port
	}

	if parsed.Scheme != "" {
		cfg.Scheme = parsed.Scheme
	}

	return cfg, nil
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

// EmbedBatch generates embeddings for multiple texts by calling Embed for
// each input in order and preserves input order in the response.
//
// vectors, _ := client.EmbedBatch(ctx, []string{"How do goroutines work?", "What is Qdrant?"})
func (o *OllamaClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := o.Embed(ctx, text)
		if err != nil {
			return nil, core.E("rag.Ollama.EmbedBatch", core.Sprintf("error embedding item %d", i), err)
		}
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
