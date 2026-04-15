package rag

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// NewOllamaEmbedder creates an Ollama client from a base endpoint URL string.
// endpoint := "http://localhost:11434"
func NewOllamaEmbedder(endpoint string, model string) (*OllamaClient, error) {
	host, port, err := parseHostPort(endpoint, 11434)
	if err != nil {
		return nil, core.E("rag.NewOllamaEmbedder", "invalid Ollama endpoint", err)
	}
	return NewOllamaClient(OllamaConfig{
		Host:  host,
		Port:  port,
		Model: model,
	})
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

func parseHostPort(endpoint string, defaultPort int) (string, int, error) {
	if endpoint == "" {
		return "localhost", defaultPort, nil
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", 0, err
	}

	host := parsed.Hostname()
	if host == "" {
		host = parsed.Path
	}
	if host == "" {
		host = "localhost"
	}

	port := defaultPort
	if parsed.Port() != "" {
		if parsedPort, err := strconv.Atoi(parsed.Port()); err == nil {
			port = parsedPort
		}
	}

	return host, port, nil
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
// each input in order.
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
			return nil, core.E(
				"rag.Ollama.EmbedBatch",
				core.Sprintf("error embedding text at index %d", i),
				err,
			)
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
