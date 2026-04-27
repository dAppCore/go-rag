package rag

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ollama/ollama/api"
)

// --- DefaultOllamaConfig tests ---

func TestOllama_DefaultOllamaConfig_Good(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultOllamaConfig()

		assertEqual(t, "localhost", cfg.Host, "default host should be localhost")
		assertEqual(t, 11434, cfg.Port, "default port should be 11434")
		assertEqual(t, "nomic-embed-text", cfg.Model, "default model should be nomic-embed-text")
	})
}

// --- NewOllamaClient tests ---

func TestOllama_NewOllamaClient_Good(t *testing.T) {
	t.Run("constructs client with default config", func(t *testing.T) {
		client, err := NewOllamaClient(DefaultOllamaConfig())

		assertNoError(t, err)
		assertNotNil(t, client)
		assertEqual(t, "nomic-embed-text", client.Model())
		assertEqual(t, uint64(768), client.EmbedDimension())
	})

	t.Run("constructs client with custom config", func(t *testing.T) {
		cfg := OllamaConfig{
			Host:  "10.0.0.1",
			Port:  8080,
			Model: "mxbai-embed-large",
		}
		client, err := NewOllamaClient(cfg)

		assertNoError(t, err)
		assertNotNil(t, client)
		assertEqual(t, "mxbai-embed-large", client.Model())
		assertEqual(t, uint64(1024), client.EmbedDimension())
	})
}

// --- EmbedDimension tests ---

func TestOllama_EmbedDimension_Good(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected uint64
	}{
		{
			name:     "nomic-embed-text returns 768",
			model:    "nomic-embed-text",
			expected: 768,
		},
		{
			name:     "mxbai-embed-large returns 1024",
			model:    "mxbai-embed-large",
			expected: 1024,
		},
		{
			name:     "all-minilm returns 384",
			model:    "all-minilm",
			expected: 384,
		},
		{
			name:     "unknown model defaults to 768",
			model:    "some-unknown-model",
			expected: 768,
		},
		{
			name:     "empty model name defaults to 768",
			model:    "",
			expected: 768,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Construct OllamaClient directly with the config to avoid needing a live server.
			// We only set the config field; client is nil but EmbedDimension does not use it.
			client := &OllamaClient{
				config: OllamaConfig{Model: tc.model},
			}

			dim := client.EmbedDimension()
			assertEqual(t, tc.expected, dim)
		})
	}
}

// --- Model tests ---

func TestOllama_Model_Good(t *testing.T) {
	t.Run("returns the configured model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "nomic-embed-text"},
		}

		assertEqual(t, "nomic-embed-text", client.Model())
	})

	t.Run("returns custom model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "custom-model"},
		}

		assertEqual(t, "custom-model", client.Model())
	})
}

// --- EmbedBatch tests ---

func TestOllama_EmbedBatch_Good(t *testing.T) {
	var requestCount int
	var capturedInput []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		body, err := io.ReadAll(r.Body)
		assertNoError(t, err)

		var req struct {
			Input string `json:"input"`
		}
		assertNoError(t, json.Unmarshal(body, &req))
		capturedInput = append(capturedInput, req.Input)

		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
			return
		}
		_, _ = w.Write([]byte(`{"embeddings":[[0.3,0.4]]}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	assertNoError(t, err)

	client := &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}

	vectors, err := client.EmbedBatch(context.Background(), []string{"first", "second"})
	assertNoError(t, err)
	assertLen(t, vectors, 2)
	assertEqual(t, []float32{0.1, 0.2}, vectors[0])
	assertEqual(t, []float32{0.3, 0.4}, vectors[1])
	assertEqual(t, 2, requestCount, "batch embedding should call Embed once per input")
	assertEqual(t, []string{"first", "second"}, capturedInput)
}

func TestOllama_EmbedBatch_Bad(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"embeddings":[[0.1,0.2]]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[]}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL)
	assertNoError(t, err)

	client := &OllamaClient{
		client: api.NewClient(baseURL, server.Client()),
		config: OllamaConfig{Model: "nomic-embed-text"},
	}

	_, err = client.EmbedBatch(context.Background(), []string{"first", "second"})
	assertError(t, err)
	assertContains(t, err.Error(), "item 1")
	assertEqual(t, 2, requestCount)
}
