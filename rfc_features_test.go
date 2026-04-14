package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkBySentences(t *testing.T) {
	chunks := ChunkBySentences("One. Two. Three.", ChunkConfig{Size: 8, Overlap: 0})

	require.Len(t, chunks, 3)
	assert.Equal(t, "One.", chunks[0].Text)
	assert.Equal(t, "Two.", chunks[1].Text)
	assert.Equal(t, "Three.", chunks[2].Text)
}

func TestChunkByParagraphs(t *testing.T) {
	text := "First paragraph.\n\nSecond paragraph."
	chunks := ChunkByParagraphs(text, ChunkConfig{Size: 100, Overlap: 0})

	require.Len(t, chunks, 2)
	assert.Equal(t, "First paragraph.", chunks[0].Text)
	assert.Equal(t, "Second paragraph.", chunks[1].Text)
}

func TestRank(t *testing.T) {
	results := []QueryResult{
		{Text: "duplicate low", Source: "a.md", ChunkIndex: 1, Score: 0.4},
		{Text: "duplicate high", Source: "a.md", ChunkIndex: 1, Score: 0.9},
		{Text: "other", Source: "b.md", ChunkIndex: 2, Score: 0.8},
	}

	ranked := Rank(results, 2)

	require.Len(t, ranked, 2)
	assert.Equal(t, "duplicate high", ranked[0].Text)
	assert.Equal(t, "other", ranked[1].Text)
}

func TestJoinResults(t *testing.T) {
	results := []QueryResult{
		{Text: "alpha"},
		{Text: "beta"},
	}

	assert.Equal(t, "alpha\n\nbeta", JoinResults(results))
}

func TestJoinResultsSearchResult(t *testing.T) {
	results := []SearchResult{
		{Text: "alpha"},
		{Text: "beta"},
	}

	assert.Equal(t, "alpha\n\nbeta", JoinResults(results))
}

func TestRankSearchResult(t *testing.T) {
	results := []SearchResult{
		{Text: "duplicate low", Source: "a.md", Index: 1, Score: 0.4},
		{Text: "duplicate high", Source: "a.md", Index: 1, Score: 0.9},
		{Text: "other", Source: "b.md", Index: 2, Score: 0.8},
	}

	ranked := Rank(results, 2)

	require.Len(t, ranked, 2)
	assert.Equal(t, "duplicate high", ranked[0].Text)
	assert.Equal(t, "other", ranked[1].Text)
}

func TestKeywordIndex(t *testing.T) {
	index := NewKeywordIndex([]Chunk{
		{Text: "Kubernetes deployment guide", Section: "Ops", Index: 0},
		{Text: "Banana bread recipe", Section: "Food", Index: 1},
	})

	hits := index.Search("kubernetes guide", 1)

	require.Len(t, hits, 1)
	assert.Equal(t, "Kubernetes deployment guide", hits[0].Text)
	assert.Equal(t, "Ops", hits[0].Section)
}

func TestEndpointConfigParsing(t *testing.T) {
	qcfg, err := qdrantConfigFromEndpoint("https://example.com:6333")
	require.NoError(t, err)
	assert.Equal(t, "example.com", qcfg.Host)
	assert.Equal(t, 6333, qcfg.Port)
	assert.True(t, qcfg.UseTLS)

	ocfg, err := ollamaConfigFromEndpoint("http://ollama.local:11435")
	require.NoError(t, err)
	assert.Equal(t, "ollama.local", ocfg.Host)
	assert.Equal(t, 11435, ocfg.Port)
	assert.Equal(t, "http", ocfg.Scheme)
}
