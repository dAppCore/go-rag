package rag

import (
	"testing"
)

func TestEndpoint_QdrantConfigFromEndpoint(t *testing.T) {
	cfg, err := qdrantConfigFromEndpoint("https://example.com:6333")
	assertNoError(t, err)
	assertEqual(t, "example.com", cfg.Host)
	assertEqual(t, 6333, cfg.Port)
	assertTrue(t, cfg.UseTLS)

	cfg, err = qdrantConfigFromEndpoint("localhost:6333")
	assertNoError(t, err)
	assertEqual(t, "localhost", cfg.Host)
	assertEqual(t, 6333, cfg.Port)
	assertFalse(t, cfg.UseTLS)
}

func TestEndpoint_OllamaConfigFromEndpoint(t *testing.T) {
	cfg, err := ollamaConfigFromEndpoint("http://ollama.local:11435")
	assertNoError(t, err)
	assertEqual(t, "ollama.local", cfg.Host)
	assertEqual(t, 11435, cfg.Port)
	assertEqual(t, "http", cfg.Scheme)

	cfg, err = ollamaConfigFromEndpoint("localhost:11434")
	assertNoError(t, err)
	assertEqual(t, "localhost", cfg.Host)
	assertEqual(t, 11434, cfg.Port)
	assertEqual(t, "http", cfg.Scheme)
}
