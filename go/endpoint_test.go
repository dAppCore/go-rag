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

// TestEndpoint_parseEndpointPort_Good — a valid in-range port string parses
// to its integer value.
func TestEndpoint_parseEndpointPort_Good(t *testing.T) {
	r := parseEndpointPort("test", "6333")
	assertEqual(t, 6333, resultValue[int](t, r))
}

// TestEndpoint_parseEndpointPort_Bad — a non-numeric port surfaces an
// "invalid port" error carrying the offending text.
func TestEndpoint_parseEndpointPort_Bad(t *testing.T) {
	r := parseEndpointPort("test", "not-a-port")
	assertError(t, r)
	assertContains(t, r.Error(), "invalid port")
}

// TestEndpoint_parseEndpointPort_Ugly — ports below 1 or above 65535 are
// rejected as out of range.
func TestEndpoint_parseEndpointPort_Ugly(t *testing.T) {
	low := parseEndpointPort("test", "0")
	assertError(t, low)
	assertContains(t, low.Error(), "out of range")

	high := parseEndpointPort("test", "70000")
	assertError(t, high)
	assertContains(t, high.Error(), "out of range")
}
