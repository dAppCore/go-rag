package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultOllamaConfig tests ---

func TestDefaultOllamaConfig(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultOllamaConfig()

		assert.Equal(t, "localhost", cfg.Host, "default host should be localhost")
		assert.Equal(t, 11434, cfg.Port, "default port should be 11434")
		assert.Equal(t, "nomic-embed-text", cfg.Model, "default model should be nomic-embed-text")
	})
}

// --- NewOllamaClient tests ---

func TestNewOllamaClient(t *testing.T) {
	t.Run("constructs client with default config", func(t *testing.T) {
		client, err := NewOllamaClient(DefaultOllamaConfig())

		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "nomic-embed-text", client.Model())
		assert.Equal(t, uint64(768), client.EmbedDimension())
	})

	t.Run("constructs client with custom config", func(t *testing.T) {
		cfg := OllamaConfig{
			Host:  "10.0.0.1",
			Port:  8080,
			Model: "mxbai-embed-large",
		}
		client, err := NewOllamaClient(cfg)

		require.NoError(t, err)
		require.NotNil(t, client)
		assert.Equal(t, "mxbai-embed-large", client.Model())
		assert.Equal(t, uint64(1024), client.EmbedDimension())
	})
}

// --- EmbedDimension tests ---

func TestEmbedDimension(t *testing.T) {
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
			assert.Equal(t, tc.expected, dim)
		})
	}
}

// --- Model tests ---

func TestModel(t *testing.T) {
	t.Run("returns the configured model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "nomic-embed-text"},
		}

		assert.Equal(t, "nomic-embed-text", client.Model())
	})

	t.Run("returns custom model name", func(t *testing.T) {
		client := &OllamaClient{
			config: OllamaConfig{Model: "custom-model"},
		}

		assert.Equal(t, "custom-model", client.Model())
	})
}
