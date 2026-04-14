package rag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- KeywordFilter tests ---

func TestKeyword_KeywordFilter_Good(t *testing.T) {
	t.Run("no keywords returns results unchanged", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Hello world.", Score: 0.9},
			{Text: "Goodbye world.", Score: 0.8},
		}

		filtered := KeywordFilter(results, nil)

		require.Len(t, filtered, 2)
		assert.Equal(t, float32(0.9), filtered[0].Score)
		assert.Equal(t, float32(0.8), filtered[1].Score)
	})

	t.Run("empty keywords returns results unchanged", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Hello world.", Score: 0.9},
		}

		filtered := KeywordFilter(results, []string{})

		require.Len(t, filtered, 1)
		assert.Equal(t, float32(0.9), filtered[0].Score)
	})

	t.Run("single keyword boosts matching result", func(t *testing.T) {
		results := []QueryResult{
			{Text: "This document is about Go programming.", Score: 0.8},
			{Text: "This document is about Python scripting.", Score: 0.9},
		}

		filtered := KeywordFilter(results, []string{"Go"})

		require.Len(t, filtered, 2)
		// Go result should be boosted by 10%: 0.8 * 1.1 = 0.88
		// Python result unchanged: 0.9
		// Python (0.9) > Go (0.88), so Python still first
		assert.Equal(t, "This document is about Python scripting.", filtered[0].Text)
		assert.InDelta(t, 0.9, filtered[0].Score, 0.001)
		assert.Equal(t, "This document is about Go programming.", filtered[1].Text)
		assert.InDelta(t, 0.88, filtered[1].Score, 0.001)
	})

	t.Run("single keyword can reorder results", func(t *testing.T) {
		results := []QueryResult{
			{Text: "General information about various topics.", Score: 0.85},
			{Text: "Detailed guide to Kubernetes deployment.", Score: 0.80},
		}

		filtered := KeywordFilter(results, []string{"kubernetes"})

		require.Len(t, filtered, 2)
		// Kubernetes result boosted: 0.80 * 1.1 = 0.88 > 0.85
		assert.Equal(t, "Detailed guide to Kubernetes deployment.", filtered[0].Text)
		assert.InDelta(t, 0.88, filtered[0].Score, 0.001)
	})

	t.Run("multiple keywords compound boost", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Go is a programming language for systems.", Score: 0.7},
			{Text: "Python is used for machine learning tasks.", Score: 0.9},
			{Text: "Go and Rust are systems programming languages.", Score: 0.6},
		}

		filtered := KeywordFilter(results, []string{"go", "systems"})

		require.Len(t, filtered, 3)
		// First result matches both: 0.7 * 1.2 = 0.84
		// Second result matches neither: 0.9
		// Third result matches both: 0.6 * 1.2 = 0.72
		// Order: Python (0.9), first Go (0.84), third Go+Rust (0.72)
		assert.Equal(t, "Python is used for machine learning tasks.", filtered[0].Text)
		assert.InDelta(t, 0.9, filtered[0].Score, 0.001)
		assert.Equal(t, "Go is a programming language for systems.", filtered[1].Text)
		assert.InDelta(t, 0.84, filtered[1].Score, 0.001)
		assert.Equal(t, "Go and Rust are systems programming languages.", filtered[2].Text)
		assert.InDelta(t, 0.72, filtered[2].Score, 0.001)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		results := []QueryResult{
			{Text: "KUBERNETES is a container orchestration platform.", Score: 0.7},
			{Text: "Docker runs containers.", Score: 0.8},
		}

		filtered := KeywordFilter(results, []string{"kubernetes"})

		require.Len(t, filtered, 2)
		// KUBERNETES matches "kubernetes" case-insensitively: 0.7 * 1.1 = 0.77
		assert.InDelta(t, 0.77, filtered[1].Score, 0.001)
		assert.Equal(t, "KUBERNETES is a container orchestration platform.", filtered[1].Text)
	})

	t.Run("no matches leaves scores unchanged", func(t *testing.T) {
		results := []QueryResult{
			{Text: "This is about cats.", Score: 0.9},
			{Text: "This is about dogs.", Score: 0.8},
		}

		filtered := KeywordFilter(results, []string{"elephants"})

		require.Len(t, filtered, 2)
		assert.Equal(t, float32(0.9), filtered[0].Score)
		assert.Equal(t, float32(0.8), filtered[1].Score)
		assert.Equal(t, "This is about cats.", filtered[0].Text)
		assert.Equal(t, "This is about dogs.", filtered[1].Text)
	})

	t.Run("empty results returns empty", func(t *testing.T) {
		filtered := KeywordFilter(nil, []string{"test"})
		assert.Empty(t, filtered)
	})
}

// --- KeywordFilterSeq tests ---

func TestKeyword_KeywordFilterSeq_Good(t *testing.T) {
	t.Run("yields boosted results via iterator", func(t *testing.T) {
		results := []QueryResult{
			{Text: "General information about various topics.", Score: 0.85},
			{Text: "Detailed guide to Kubernetes deployment.", Score: 0.80},
		}

		var collected []QueryResult
		for r := range KeywordFilterSeq(results, []string{"kubernetes"}) {
			collected = append(collected, r)
		}

		require.Len(t, collected, 2)
		// Kubernetes result boosted: 0.80 * 1.1 = 0.88 > 0.85
		assert.Equal(t, "Detailed guide to Kubernetes deployment.", collected[0].Text)
		assert.InDelta(t, 0.88, collected[0].Score, 0.001)
	})

	t.Run("empty results yields nothing", func(t *testing.T) {
		count := 0
		for range KeywordFilterSeq(nil, []string{"test"}) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("early break stops iteration", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result.", Score: 0.9},
			{Text: "Second result.", Score: 0.8},
			{Text: "Third result.", Score: 0.7},
		}

		var first QueryResult
		for r := range KeywordFilterSeq(results, nil) {
			first = r
			break
		}
		assert.Equal(t, "First result.", first.Text)
	})
}

// --- extractKeywords tests ---

func TestKeyword_ExtractKeywords_Good(t *testing.T) {
	t.Run("extracts words 3+ characters", func(t *testing.T) {
		keywords := extractKeywords("how do I use Go modules")
		assert.Contains(t, keywords, "how")
		assert.Contains(t, keywords, "use")
		assert.Contains(t, keywords, "modules")
		// "do" and "I" are too short
		assert.NotContains(t, keywords, "do")
		assert.NotContains(t, keywords, "i")
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		keywords := extractKeywords("")
		assert.Empty(t, keywords)
	})

	t.Run("all short words returns empty", func(t *testing.T) {
		keywords := extractKeywords("I am a")
		assert.Empty(t, keywords)
	})

	t.Run("keywords are lowercased", func(t *testing.T) {
		keywords := extractKeywords("Kubernetes Deployment")
		assert.Contains(t, keywords, "kubernetes")
		assert.Contains(t, keywords, "deployment")
	})
}

// --- KeywordIndex tests ---

func TestKeyword_KeywordIndex_Good(t *testing.T) {
	t.Run("indexes chunks and reports length", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "authentication guide for setup", Section: "Auth", Index: 0},
			{Text: "deployment guide for kubernetes", Section: "Deploy", Index: 1},
			{Text: "general overview of the platform", Section: "Intro", Index: 2},
		}
		idx := NewKeywordIndex(chunks)
		assert.Equal(t, 3, idx.Len())
	})

	t.Run("search returns matching chunks ranked by TF-IDF", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "authentication guide explains setup steps", Section: "Auth", Index: 0},
			{Text: "deployment guide explains kubernetes install", Section: "Deploy", Index: 1},
			{Text: "general overview of the platform features", Section: "Intro", Index: 2},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("authentication setup", 5)

		require.NotEmpty(t, results)
		// Top result must contain both query terms.
		assert.Contains(t, results[0].Text, "authentication")
		assert.Contains(t, results[0].Text, "setup")
		assert.Equal(t, "Auth", results[0].Section)
		assert.Equal(t, 0, results[0].ChunkIndex)
		assert.Greater(t, results[0].Score, float32(0))
	})

	t.Run("rare terms outrank common terms", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "guide guide guide guide kubernetes", Index: 0},
			{Text: "guide guide guide guide authentication", Index: 1},
		}
		idx := NewKeywordIndex(chunks)
		// "guide" appears in every chunk (IDF=0), kubernetes is rarer.
		results := idx.Search("kubernetes guide", 5)

		require.NotEmpty(t, results)
		assert.Equal(t, 0, results[0].ChunkIndex)
	})

	t.Run("topK limits result count", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "matching term foo here", Index: 0},
			{Text: "matching term foo here", Index: 1},
			{Text: "matching term foo here", Index: 2},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("foo", 2)
		assert.LessOrEqual(t, len(results), 2)
	})

	t.Run("non-matching query returns empty", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "kubernetes deployment guide", Index: 0},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("elephants", 5)
		assert.Empty(t, results)
	})

	t.Run("score is positive for matching chunks", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "alpha beta gamma", Index: 0},
			{Text: "delta epsilon zeta", Index: 1},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("alpha", 5)
		require.Len(t, results, 1)
		assert.Greater(t, results[0].Score, float32(0))
	})
}

func TestKeyword_KeywordIndex_Bad(t *testing.T) {
	t.Run("nil receiver Search returns nil", func(t *testing.T) {
		var idx *KeywordIndex
		results := idx.Search("anything", 5)
		assert.Nil(t, results)
	})

	t.Run("nil receiver Len returns zero", func(t *testing.T) {
		var idx *KeywordIndex
		assert.Equal(t, 0, idx.Len())
	})

	t.Run("empty chunks yields empty index", func(t *testing.T) {
		idx := NewKeywordIndex(nil)
		assert.Equal(t, 0, idx.Len())
		assert.Empty(t, idx.Search("query", 5))
	})

	t.Run("negative topK still yields full set", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "match here", Index: 0},
			{Text: "match here", Index: 1},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("match", -1)
		assert.Len(t, results, 2)
	})

	t.Run("topK larger than result set returns all matches", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "kubernetes guide", Index: 0},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("kubernetes", 100)
		assert.Len(t, results, 1)
	})
}

func TestKeyword_KeywordIndex_Ugly(t *testing.T) {
	t.Run("empty query returns nil", func(t *testing.T) {
		chunks := []Chunk{{Text: "some chunk text", Index: 0}}
		idx := NewKeywordIndex(chunks)
		assert.Nil(t, idx.Search("", 5))
	})

	t.Run("query with only short tokens returns nil", func(t *testing.T) {
		chunks := []Chunk{{Text: "some chunk text", Index: 0}}
		idx := NewKeywordIndex(chunks)
		// Every token is shorter than 3 chars — all dropped.
		assert.Nil(t, idx.Search("a b c", 5))
	})

	t.Run("empty chunk text is indexed but unreachable", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "", Index: 0},
			{Text: "reachable content", Index: 1},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("reachable", 5)
		require.Len(t, results, 1)
		assert.Equal(t, 1, results[0].ChunkIndex)
	})

	t.Run("punctuation and case are normalised", func(t *testing.T) {
		chunks := []Chunk{
			{Text: "Kubernetes! Is; Fun.", Index: 0},
		}
		idx := NewKeywordIndex(chunks)
		results := idx.Search("KUBERNETES", 5)
		require.NotEmpty(t, results)
	})

	t.Run("repeated query terms do not inflate score", func(t *testing.T) {
		chunks := []Chunk{{Text: "alpha beta gamma", Index: 0}}
		idx := NewKeywordIndex(chunks)
		single := idx.Search("alpha", 5)
		repeated := idx.Search("alpha alpha alpha", 5)
		require.Len(t, single, 1)
		require.Len(t, repeated, 1)
		assert.InDelta(t, single[0].Score, repeated[0].Score, 0.0001)
	})
}

// --- Query with Keywords integration ---

func TestKeyword_QueryKeywords_Good(t *testing.T) {
	t.Run("keywords flag enables keyword boosting", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["test-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "General overview of the platform.", "source": "a.md",
				"section": "", "category": "docs", "chunk_index": 0,
			}},
			{ID: "2", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Guide to deploying with Kubernetes containers.", "source": "b.md",
				"section": "", "category": "docs", "chunk_index": 1,
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Limit = 10
		cfg.Threshold = 0.0
		cfg.Keywords = true

		results, err := Query(context.Background(), store, embedder, "kubernetes containers", cfg)

		require.NoError(t, err)
		require.Len(t, results, 2)
		// The second result (score 0.9 from mock) matches two keywords,
		// boosted to 0.9 * 1.2 = 1.08, so it should be first.
		assert.Equal(t, "Guide to deploying with Kubernetes containers.", results[0].Text)
	})

	t.Run("keywords false does not boost", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["test-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "First result text.", "source": "a.md",
				"section": "", "category": "docs", "chunk_index": 0,
			}},
			{ID: "2", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Second result text with keywords.", "source": "b.md",
				"section": "", "category": "docs", "chunk_index": 1,
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Limit = 10
		cfg.Threshold = 0.0
		cfg.Keywords = false

		results, err := Query(context.Background(), store, embedder, "keywords", cfg)

		require.NoError(t, err)
		require.Len(t, results, 2)
		// Without keywords, original order preserved (first has higher score)
		assert.Equal(t, "First result text.", results[0].Text)
	})
}
