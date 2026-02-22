package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- DefaultQueryConfig tests ---

func TestDefaultQueryConfig(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultQueryConfig()

		assert.Equal(t, "hostuk-docs", cfg.Collection, "default collection should be hostuk-docs")
		assert.Equal(t, uint64(5), cfg.Limit, "default limit should be 5")
		assert.Equal(t, float32(0.5), cfg.Threshold, "default threshold should be 0.5")
		assert.Empty(t, cfg.Category, "default category should be empty")
	})
}

// --- FormatResultsText tests ---

func TestFormatResultsText(t *testing.T) {
	t.Run("empty results returns no-results message", func(t *testing.T) {
		result := FormatResultsText(nil)
		assert.Equal(t, "No results found.", result)
	})

	t.Run("empty slice returns no-results message", func(t *testing.T) {
		result := FormatResultsText([]QueryResult{})
		assert.Equal(t, "No results found.", result)
	})

	t.Run("single result with all fields", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Some relevant text about Go.",
				Source:   "docs/go-intro.md",
				Section:  "Introduction",
				Category: "documentation",
				Score:    0.95,
			},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "Result 1")
		assert.Contains(t, output, "score: 0.95")
		assert.Contains(t, output, "Source: docs/go-intro.md")
		assert.Contains(t, output, "Section: Introduction")
		assert.Contains(t, output, "Category: documentation")
		assert.Contains(t, output, "Some relevant text about Go.")
	})

	t.Run("section omitted when empty", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "No section here.",
				Source:   "test.md",
				Section:  "",
				Category: "docs",
				Score:    0.80,
			},
		}

		output := FormatResultsText(results)

		assert.NotContains(t, output, "Section:")
	})

	t.Run("multiple results numbered correctly", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result.", Source: "a.md", Category: "docs", Score: 0.90},
			{Text: "Second result.", Source: "b.md", Category: "docs", Score: 0.85},
			{Text: "Third result.", Source: "c.md", Category: "docs", Score: 0.80},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "Result 1")
		assert.Contains(t, output, "Result 2")
		assert.Contains(t, output, "Result 3")
		// Verify ordering: first result appears before second
		idx1 := strings.Index(output, "Result 1")
		idx2 := strings.Index(output, "Result 2")
		idx3 := strings.Index(output, "Result 3")
		assert.Less(t, idx1, idx2)
		assert.Less(t, idx2, idx3)
	})

	t.Run("score formatted to two decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Test.", Source: "s.md", Category: "c", Score: 0.123456},
		}

		output := FormatResultsText(results)

		assert.Contains(t, output, "score: 0.12")
	})
}

// --- FormatResultsContext tests ---

func TestFormatResultsContext(t *testing.T) {
	t.Run("empty results returns empty string", func(t *testing.T) {
		result := FormatResultsContext(nil)
		assert.Equal(t, "", result)
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		result := FormatResultsContext([]QueryResult{})
		assert.Equal(t, "", result)
	})

	t.Run("wraps output in retrieved_context tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Hello world.", Source: "test.md", Section: "Intro", Category: "docs", Score: 0.9},
		}

		output := FormatResultsContext(results)

		assert.True(t, strings.HasPrefix(output, "<retrieved_context>\n"),
			"output should start with <retrieved_context> tag")
		assert.True(t, strings.HasSuffix(output, "</retrieved_context>"),
			"output should end with </retrieved_context> tag")
	})

	t.Run("wraps each result in document tags with attributes", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Content here.",
				Source:   "file.md",
				Section:  "My Section",
				Category: "documentation",
				Score:    0.88,
			},
		}

		output := FormatResultsContext(results)

		assert.Contains(t, output, `source="file.md"`)
		assert.Contains(t, output, `section="My Section"`)
		assert.Contains(t, output, `category="documentation"`)
		assert.Contains(t, output, "Content here.")
		assert.Contains(t, output, "</document>")
	})

	t.Run("escapes XML special characters in attributes and text", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     `Text with <tags> & "quotes" in it.`,
				Source:   `path/with<special>&chars.md`,
				Section:  `Section "One"`,
				Category: "docs",
				Score:    0.75,
			},
		}

		output := FormatResultsContext(results)

		// The html.EscapeString function escapes <, >, &, " and '
		assert.Contains(t, output, "&lt;tags&gt;")
		assert.Contains(t, output, "&amp;")
		assert.Contains(t, output, "&#34;quotes&#34;")
		// Source attribute should also be escaped
		assert.Contains(t, output, "path/with&lt;special&gt;&amp;chars.md")
	})

	t.Run("multiple results each wrapped in document tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "", Category: "docs", Score: 0.9},
			{Text: "Second.", Source: "b.md", Section: "", Category: "docs", Score: 0.8},
		}

		output := FormatResultsContext(results)

		// Count document tags
		assert.Equal(t, 2, strings.Count(output, "<document "))
		assert.Equal(t, 2, strings.Count(output, "</document>"))
	})
}

// --- FormatResultsJSON tests ---

func TestFormatResultsJSON(t *testing.T) {
	t.Run("empty results returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON(nil)
		assert.Equal(t, "[]", result)
	})

	t.Run("empty slice returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON([]QueryResult{})
		assert.Equal(t, "[]", result)
	})

	t.Run("single result produces valid JSON", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Test content.",
				Source:   "test.md",
				Section:  "Intro",
				Category: "docs",
				Score:    0.9234,
			},
		}

		output := FormatResultsJSON(results)

		// Verify it parses as valid JSON
		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON")
		require.Len(t, parsed, 1)

		assert.Equal(t, "test.md", parsed[0]["source"])
		assert.Equal(t, "Intro", parsed[0]["section"])
		assert.Equal(t, "docs", parsed[0]["category"])
		assert.Equal(t, "Test content.", parsed[0]["text"])
		// Score is formatted to 4 decimal places
		assert.InDelta(t, 0.9234, parsed[0]["score"], 0.0001)
	})

	t.Run("multiple results produce valid JSON array", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "A", Category: "docs", Score: 0.95},
			{Text: "Second.", Source: "b.md", Section: "B", Category: "code", Score: 0.80},
			{Text: "Third.", Source: "c.md", Section: "C", Category: "task", Score: 0.70},
		}

		output := FormatResultsJSON(results)

		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON")
		require.Len(t, parsed, 3)

		assert.Equal(t, "First.", parsed[0]["text"])
		assert.Equal(t, "Second.", parsed[1]["text"])
		assert.Equal(t, "Third.", parsed[2]["text"])
	})

	t.Run("special characters are JSON-escaped in text", func(t *testing.T) {
		results := []QueryResult{
			{
				Text:     "Line one\nLine two\twith tab and \"quotes\"",
				Source:   "special.md",
				Section:  "",
				Category: "docs",
				Score:    0.5,
			},
		}

		output := FormatResultsJSON(results)

		var parsed []map[string]any
		err := json.Unmarshal([]byte(output), &parsed)
		require.NoError(t, err, "output should be valid JSON even with special characters")
		assert.Equal(t, "Line one\nLine two\twith tab and \"quotes\"", parsed[0]["text"])
	})

	t.Run("score formatted to four decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "T.", Source: "s.md", Section: "", Category: "c", Score: 0.123456789},
		}

		output := FormatResultsJSON(results)

		assert.Contains(t, output, "0.1235")
	})
}

// --- Query function tests with mocks ---

func TestQuery(t *testing.T) {
	t.Run("generates embedding for query text", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"

		_, err := Query(context.Background(), store, embedder, "what is Go?", cfg)

		require.NoError(t, err)
		assert.Equal(t, 1, embedder.embedCallCount())
		assert.Equal(t, "what is Go?", embedder.embedCalls[0])
	})

	t.Run("search is called with correct parameters", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "my-docs"
		cfg.Limit = 3

		_, err := Query(context.Background(), store, embedder, "test query", cfg)

		require.NoError(t, err)
		assert.Equal(t, 1, store.searchCallCount())

		call := store.searchCalls[0]
		assert.Equal(t, "my-docs", call.Collection)
		assert.Equal(t, uint64(3), call.Limit)
		assert.Len(t, call.Vector, 768) // Vector should be 768 dimensions
		assert.Nil(t, call.Filter)      // No category filter
	})

	t.Run("category filter is passed to search", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Category = "documentation"

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		call := store.searchCalls[0]
		assert.Equal(t, map[string]string{"category": "documentation"}, call.Filter)
	})

	t.Run("returns results above threshold", func(t *testing.T) {
		store := newMockVectorStore()
		// Pre-populate store with points
		store.points["test-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "high score", "source": "a.md", "section": "S", "category": "docs", "chunk_index": 0,
			}},
			{ID: "2", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "mid score", "source": "b.md", "section": "S", "category": "docs", "chunk_index": 1,
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Limit = 10
		cfg.Threshold = 0.95 // Only the first result (score 1.0) should pass; second gets 0.9

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "high score", results[0].Text)
		assert.Equal(t, "a.md", results[0].Source)
		assert.Equal(t, "S", results[0].Section)
		assert.Equal(t, "docs", results[0].Category)
	})

	t.Run("empty results when nothing above threshold", func(t *testing.T) {
		store := newMockVectorStore()
		// No points stored — search returns empty
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "empty-col"
		cfg.Threshold = 0.5

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("empty results when store has no matching points", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "no-data"

		results, err := Query(context.Background(), store, embedder, "test query", cfg)

		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("embedder failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = fmt.Errorf("ollama down")

		cfg := DefaultQueryConfig()

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error generating query embedding")
		// Search should not be called if embedding fails
		assert.Equal(t, 0, store.searchCallCount())
	})

	t.Run("search failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = fmt.Errorf("qdrant timeout")
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error searching")
	})

	t.Run("extracts all payload fields correctly", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["test-col"] = []Point{
			{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{
				"text":        "Full payload test.",
				"source":      "docs/guide.md",
				"section":     "Getting Started",
				"category":    "documentation",
				"chunk_index": 5,
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Limit = 10
		cfg.Threshold = 0.0

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		require.Len(t, results, 1)

		r := results[0]
		assert.Equal(t, "Full payload test.", r.Text)
		assert.Equal(t, "docs/guide.md", r.Source)
		assert.Equal(t, "Getting Started", r.Section)
		assert.Equal(t, "documentation", r.Category)
		assert.Equal(t, 5, r.ChunkIndex)
	})

	t.Run("handles int64 chunk_index from Qdrant", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["test-col"] = []Point{
			{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{
				"text":        "Test.",
				"source":      "a.md",
				"section":     "",
				"category":    "docs",
				"chunk_index": int64(42),
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Threshold = 0.0

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 42, results[0].ChunkIndex)
	})

	t.Run("handles float64 chunk_index from JSON", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["test-col"] = []Point{
			{ID: "p1", Vector: []float32{0.1}, Payload: map[string]any{
				"text":        "Test.",
				"source":      "a.md",
				"section":     "",
				"category":    "docs",
				"chunk_index": float64(7),
			}},
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Threshold = 0.0

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, 7, results[0].ChunkIndex)
	})

	t.Run("results respect limit", func(t *testing.T) {
		store := newMockVectorStore()
		// Add many points
		for i := range 10 {
			store.points["test-col"] = append(store.points["test-col"], Point{
				ID:     fmt.Sprintf("p%d", i),
				Vector: []float32{0.1},
				Payload: map[string]any{
					"text":        fmt.Sprintf("Result %d", i),
					"source":      "doc.md",
					"section":     "",
					"category":    "docs",
					"chunk_index": i,
				},
			})
		}
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Limit = 3
		cfg.Threshold = 0.0

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		require.NoError(t, err)
		assert.Len(t, results, 3)
	})
}
