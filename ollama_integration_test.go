//go:build rag

package rag

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfOllamaUnavailable skips the test if Ollama is not reachable on the
// default HTTP port.
func skipIfOllamaUnavailable(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:11434", 2*time.Second)
	if err != nil {
		t.Skip("Ollama not available on localhost:11434 — skipping integration test")
	}
	_ = conn.Close()
}

func TestOllama_Integration_Ugly(t *testing.T) {
	skipIfOllamaUnavailable(t)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	require.NoError(t, err, "failed to create Ollama client")

	ctx := context.Background()

	t.Run("verify model is available", func(t *testing.T) {
		err := client.VerifyModel(ctx)
		require.NoError(t, err, "nomic-embed-text model should be available")
	})

	t.Run("embed single text returns correct dimension", func(t *testing.T) {
		vec, err := client.Embed(ctx, "The quick brown fox jumps over the lazy dog.")
		require.NoError(t, err, "embedding should succeed")
		require.NotEmpty(t, vec, "embedding vector should not be empty")

		expectedDim := client.EmbedDimension()
		assert.Equal(t, int(expectedDim), len(vec),
			"embedding dimension should match EmbedDimension() for nomic-embed-text (768)")
	})

	t.Run("embed batch returns correct number of vectors", func(t *testing.T) {
		texts := []string{
			"Go is a statically typed programming language.",
			"Rust prioritises memory safety without garbage collection.",
			"Python is popular for data science and machine learning.",
		}

		vectors, err := client.EmbedBatch(ctx, texts)
		require.NoError(t, err, "batch embedding should succeed")
		require.Len(t, vectors, 3, "should return one vector per input text")

		expectedDim := int(client.EmbedDimension())
		for i, vec := range vectors {
			assert.Len(t, vec, expectedDim,
				"vector %d should have dimension %d", i, expectedDim)
		}
	})

	t.Run("embedding consistency — same text produces identical vectors", func(t *testing.T) {
		text := "Deterministic embedding test."

		vec1, err := client.Embed(ctx, text)
		require.NoError(t, err)

		vec2, err := client.Embed(ctx, text)
		require.NoError(t, err)

		require.Equal(t, len(vec1), len(vec2), "vectors should have same length")
		for i := range vec1 {
			assert.Equal(t, vec1[i], vec2[i],
				"vectors should be identical at index %d — same input must produce same output", i)
		}
	})

	t.Run("dimension matches config — EmbedDimension equals actual embedding size", func(t *testing.T) {
		// EmbedDimension is a pure lookup, but here we verify it matches reality
		declaredDim := client.EmbedDimension()
		assert.Equal(t, uint64(768), declaredDim,
			"nomic-embed-text should declare 768 dimensions")

		vec, err := client.Embed(ctx, "dimension verification")
		require.NoError(t, err)
		assert.Equal(t, int(declaredDim), len(vec),
			"actual embedding length should match declared dimension")
	})

	t.Run("model name returns configured model", func(t *testing.T) {
		assert.Equal(t, "nomic-embed-text", client.Model(),
			"Model() should return the configured model name")
	})

	t.Run("different texts produce different embeddings", func(t *testing.T) {
		vec1, err := client.Embed(ctx, "Qdrant is a vector database.")
		require.NoError(t, err)

		vec2, err := client.Embed(ctx, "Banana bread recipe with walnuts.")
		require.NoError(t, err)

		// Check that the vectors differ in at least some positions
		differ := false
		for i := range vec1 {
			if vec1[i] != vec2[i] {
				differ = true
				break
			}
		}
		assert.True(t, differ, "semantically different texts should produce different vectors")
	})

	t.Run("embedding vectors contain non-zero values", func(t *testing.T) {
		vec, err := client.Embed(ctx, "Non-zero embedding check.")
		require.NoError(t, err)

		hasNonZero := false
		for _, v := range vec {
			if v != 0.0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "embedding should contain at least one non-zero value")
	})

	t.Run("empty string can be embedded without error", func(t *testing.T) {
		// Ollama may or may not accept empty strings — this test documents the behaviour.
		vec, err := client.Embed(ctx, "")
		if err == nil {
			// If it succeeds, the dimension should still be correct
			assert.Equal(t, int(client.EmbedDimension()), len(vec))
		}
		// If it errors, that is also acceptable — we just document it
	})
}
