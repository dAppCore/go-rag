package rag

import (
	"testing"
)

func TestChunkBySentences(t *testing.T) {
	chunks := ChunkBySentences("One. Two. Three.", ChunkConfig{Size: 8, Overlap: 0})

	assertLen(t, chunks, 3)
	assertEqual(t, "One.", chunks[0].Text)
	assertEqual(t, "Two.", chunks[1].Text)
	assertEqual(t, "Three.", chunks[2].Text)
}

func TestChunkByParagraphs(t *testing.T) {
	text := "First paragraph.\n\nSecond paragraph."
	chunks := ChunkByParagraphs(text, ChunkConfig{Size: 100, Overlap: 0})

	assertLen(t, chunks, 1)
	assertContains(t, chunks[0].Text, "First paragraph.")
	assertContains(t, chunks[0].Text, "Second paragraph.")
}

func TestRank(t *testing.T) {
	results := []QueryResult{
		{Text: "duplicate low", Source: "a.md", ChunkIndex: 1, Score: 0.4},
		{Text: "duplicate high", Source: "a.md", ChunkIndex: 1, Score: 0.9},
		{Text: "other", Source: "b.md", ChunkIndex: 2, Score: 0.8},
	}

	ranked := Rank(results, 2)

	assertLen(t, ranked, 2)
	assertEqual(t, "duplicate high", ranked[0].Text)
	assertEqual(t, "other", ranked[1].Text)
}

func TestJoinResults(t *testing.T) {
	results := []QueryResult{
		{Text: "alpha"},
		{Text: "beta"},
	}

	assertEqual(t, "alpha\n\nbeta", JoinResults(results))
}

func TestJoinResultsSearchResult(t *testing.T) {
	results := []SearchResult{
		{Text: "alpha"},
		{Text: "beta"},
	}

	assertEqual(t, "alpha\n\nbeta", JoinResults(results))
}

func TestRankSearchResult(t *testing.T) {
	results := []SearchResult{
		{Text: "duplicate low", Source: "a.md", Index: 1, Score: 0.4},
		{Text: "duplicate high", Source: "a.md", Index: 1, Score: 0.9},
		{Text: "other", Source: "b.md", Index: 2, Score: 0.8},
	}

	ranked := Rank(results, 2)

	assertLen(t, ranked, 2)
	assertEqual(t, "duplicate high", ranked[0].Text)
	assertEqual(t, "other", ranked[1].Text)
}

func TestKeywordIndex(t *testing.T) {
	index := NewKeywordIndex([]Chunk{
		{Text: "Kubernetes deployment guide", Section: "Ops", Index: 0},
		{Text: "Banana bread recipe", Section: "Food", Index: 1},
	})

	hits := index.Search("kubernetes guide", 1)

	assertLen(t, hits, 1)
	assertEqual(t, "Kubernetes deployment guide", hits[0].Text)
	assertEqual(t, "Ops", hits[0].Section)
}

func TestEndpointConfigParsing(t *testing.T) {
	qcfg, err := qdrantConfigFromEndpoint("https://example.com:6333")
	assertNoError(t, err)
	assertEqual(t, "example.com", qcfg.Host)
	assertEqual(t, 6333, qcfg.Port)
	assertTrue(t, qcfg.UseTLS)

	ocfg, err := ollamaConfigFromEndpoint("http://ollama.local:11435")
	assertNoError(t, err)
	assertEqual(t, "ollama.local", ocfg.Host)
	assertEqual(t, 11435, ocfg.Port)
	assertEqual(t, "http", ocfg.Scheme)
}
