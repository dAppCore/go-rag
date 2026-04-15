package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_QdrantConfigFromEndpoint(t *testing.T) {
	cfg, err := qdrantConfigFromEndpoint("https://example.com:6333")
	require.NoError(t, err)
	assert.Equal(t, "example.com", cfg.Host)
	assert.Equal(t, 6333, cfg.Port)
	assert.True(t, cfg.UseTLS)

	cfg, err = qdrantConfigFromEndpoint("localhost:6333")
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 6333, cfg.Port)
	assert.False(t, cfg.UseTLS)
}

func TestEndpoint_OllamaConfigFromEndpoint(t *testing.T) {
	cfg, err := ollamaConfigFromEndpoint("http://ollama.local:11435")
	require.NoError(t, err)
	assert.Equal(t, "ollama.local", cfg.Host)
	assert.Equal(t, 11435, cfg.Port)
	assert.Equal(t, "http", cfg.Scheme)

	cfg, err = ollamaConfigFromEndpoint("localhost:11434")
	require.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 11434, cfg.Port)
	assert.Equal(t, "http", cfg.Scheme)
}
