package rag

import (
	"testing"
)

func TestEndpoint_QdrantConfigFromEndpoint(t *testing.T) {
	r := qdrantConfigFromEndpoint("https://example.com:6333")
	cfg := resultValue[QdrantConfig](t, r)
	assertEqual(t, "example.com", cfg.Host)
	assertEqual(t, 6333, cfg.Port)
	assertTrue(t, cfg.UseTLS)

	r = qdrantConfigFromEndpoint("localhost:6333")
	cfg = resultValue[QdrantConfig](t, r)
	assertEqual(t, "localhost", cfg.Host)
	assertEqual(t, 6333, cfg.Port)
	assertFalse(t, cfg.UseTLS)
}

func TestEndpoint_OllamaConfigFromEndpoint(t *testing.T) {
	r := ollamaConfigFromEndpoint("http://ollama.local:11435")
	cfg := resultValue[OllamaConfig](t, r)
	assertEqual(t, "ollama.local", cfg.Host)
	assertEqual(t, 11435, cfg.Port)
	assertEqual(t, "http", cfg.Scheme)

	r = ollamaConfigFromEndpoint("localhost:11434")
	cfg = resultValue[OllamaConfig](t, r)
	assertEqual(t, "localhost", cfg.Host)
	assertEqual(t, 11434, cfg.Port)
	assertEqual(t, "http", cfg.Scheme)
}
