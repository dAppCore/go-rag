package rag

import (
	"context"
	"testing"

	"dappco.re/go/core"
)

// --- DefaultQueryConfig tests ---

func TestQuery_DefaultQueryConfig_Good(t *testing.T) {
	t.Run("returns expected default values", func(t *testing.T) {
		cfg := DefaultQueryConfig()

		assertEqual(t, "hostuk-docs", cfg.Collection, "default collection should be hostuk-docs")
		assertEqual(t, uint64(5), cfg.Limit, "default limit should be 5")
		assertEqual(t, float32(0.5), cfg.Threshold, "default threshold should be 0.5")
		assertEmpty(t, cfg.Category, "default category should be empty")
	})
}

// --- FormatResultsText tests ---

func TestQuery_FormatResultsText_Good(t *testing.T) {
	t.Run("empty results returns no-results message", func(t *testing.T) {
		result := FormatResultsText(nil)
		assertEqual(t, "No results found.", result)
	})

	t.Run("empty slice returns no-results message", func(t *testing.T) {
		result := FormatResultsText([]QueryResult{})
		assertEqual(t, "No results found.", result)
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

		assertContains(t, output, "Result 1")
		assertContains(t, output, "score: 0.95")
		assertContains(t, output, "Source: docs/go-intro.md")
		assertContains(t, output, "Section: Introduction")
		assertContains(t, output, "Category: documentation")
		assertContains(t, output, "Some relevant text about Go.")
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

		assertNotContains(t, output, "Section:")
	})

	t.Run("multiple results numbered correctly", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result.", Source: "a.md", Category: "docs", Score: 0.90},
			{Text: "Second result.", Source: "b.md", Category: "docs", Score: 0.85},
			{Text: "Third result.", Source: "c.md", Category: "docs", Score: 0.80},
		}

		output := FormatResultsText(results)

		assertContains(t, output, "Result 1")
		assertContains(t, output, "Result 2")
		assertContains(t, output, "Result 3")
		// Verify ordering: first result appears before second
		idx1 := indexOf(output, "Result 1")
		idx2 := indexOf(output, "Result 2")
		idx3 := indexOf(output, "Result 3")
		assertLess(t, idx1, idx2)
		assertLess(t, idx2, idx3)
	})

	t.Run("score formatted to two decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Test.", Source: "s.md", Category: "c", Score: 0.123456},
		}

		output := FormatResultsText(results)

		assertContains(t, output, "score: 0.12")
	})
}

// --- FormatResultsContext tests ---

func TestQuery_FormatResultsContext_Good(t *testing.T) {
	t.Run("empty results returns empty string", func(t *testing.T) {
		result := FormatResultsContext(nil)
		assertEqual(t, "", result)
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		result := FormatResultsContext([]QueryResult{})
		assertEqual(t, "", result)
	})

	t.Run("wraps output in retrieved_context tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "Hello world.", Source: "test.md", Section: "Intro", Category: "docs", Score: 0.9},
		}

		output := FormatResultsContext(results)

		assertTrue(t, core.HasPrefix(output, "<retrieved_context>\n"),
			"output should start with <retrieved_context> tag")
		assertTrue(t, core.HasSuffix(output, "</retrieved_context>"),
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

		assertContains(t, output, `source="file.md"`)
		assertContains(t, output, `section="My Section"`)
		assertContains(t, output, `category="documentation"`)
		assertContains(t, output, "Content here.")
		assertContains(t, output, "</document>")
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
		assertContains(t, output, "&lt;tags&gt;")
		assertContains(t, output, "&amp;")
		assertContains(t, output, "&#34;quotes&#34;")
		// Source attribute should also be escaped
		assertContains(t, output, "path/with&lt;special&gt;&amp;chars.md")
	})

	t.Run("multiple results each wrapped in document tags", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "", Category: "docs", Score: 0.9},
			{Text: "Second.", Source: "b.md", Section: "", Category: "docs", Score: 0.8},
		}

		output := FormatResultsContext(results)

		// Count document tags
		assertEqual(t, 2, len(core.Split(output, "<document "))-1)
		assertEqual(t, 2, len(core.Split(output, "</document>"))-1)
	})
}

// --- FormatResultsJSON tests ---

func TestQuery_FormatResultsJSON_Good(t *testing.T) {
	t.Run("empty results returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON(nil)
		assertEqual(t, "[]", result)
	})

	t.Run("empty slice returns empty JSON array", func(t *testing.T) {
		result := FormatResultsJSON([]QueryResult{})
		assertEqual(t, "[]", result)
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
		result := core.JSONUnmarshalString(output, &parsed)
		assertTrue(t, result.OK, "output should be valid JSON")
		assertLen(t, parsed, 1)

		assertEqual(t, "test.md", parsed[0]["source"])
		assertEqual(t, "Intro", parsed[0]["section"])
		assertEqual(t, "docs", parsed[0]["category"])
		assertEqual(t, "Test content.", parsed[0]["text"])
		// Score is formatted to 4 decimal places
		assertInDelta(t, 0.9234, parsed[0]["score"], 0.0001)
	})

	t.Run("multiple results produce valid JSON array", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First.", Source: "a.md", Section: "A", Category: "docs", Score: 0.95},
			{Text: "Second.", Source: "b.md", Section: "B", Category: "code", Score: 0.80},
			{Text: "Third.", Source: "c.md", Section: "C", Category: "task", Score: 0.70},
		}

		output := FormatResultsJSON(results)

		var parsed []map[string]any
		result := core.JSONUnmarshalString(output, &parsed)
		assertTrue(t, result.OK, "output should be valid JSON")
		assertLen(t, parsed, 3)

		assertEqual(t, "First.", parsed[0]["text"])
		assertEqual(t, "Second.", parsed[1]["text"])
		assertEqual(t, "Third.", parsed[2]["text"])
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
		result := core.JSONUnmarshalString(output, &parsed)
		assertTrue(t, result.OK, "output should be valid JSON even with special characters")
		assertEqual(t, "Line one\nLine two\twith tab and \"quotes\"", parsed[0]["text"])
	})

	t.Run("score formatted to four decimal places", func(t *testing.T) {
		results := []QueryResult{
			{Text: "T.", Source: "s.md", Section: "", Category: "c", Score: 0.123456789},
		}

		output := FormatResultsJSON(results)

		assertContains(t, output, "0.1235")
	})
}

// --- Query function tests with mocks ---

func TestQuery_Query_Good(t *testing.T) {
	t.Run("generates embedding for query text", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"

		_, err := Query(context.Background(), store, embedder, "what is Go?", cfg)

		assertNoError(t, err)
		assertEqual(t, 1, embedder.embedCallCount())
		assertEqual(t, "what is Go?", embedder.embedCalls[0])
	})

	t.Run("search is called with correct parameters", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "my-docs"
		cfg.Limit = 3

		_, err := Query(context.Background(), store, embedder, "test query", cfg)

		assertNoError(t, err)
		assertEqual(t, 1, store.searchCallCount())

		call := store.searchCalls[0]
		assertEqual(t, "my-docs", call.Collection)
		assertEqual(t, uint64(3), call.Limit)
		assertLen(t, call.Vector, 768) // Vector should be 768 dimensions
		assertNil(t, call.Filter)      // No category filter
	})

	t.Run("category filter is passed to search", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "test-col"
		cfg.Category = "documentation"

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		assertNoError(t, err)
		call := store.searchCalls[0]
		assertEqual(t, map[string]string{"category": "documentation"}, call.Filter)
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

		assertNoError(t, err)
		assertLen(t, results, 1)
		assertEqual(t, "high score", results[0].Text)
		assertEqual(t, "a.md", results[0].Source)
		assertEqual(t, "S", results[0].Section)
		assertEqual(t, "docs", results[0].Category)
	})

	t.Run("empty results when nothing above threshold", func(t *testing.T) {
		store := newMockVectorStore()
		// No points stored — search returns empty
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "empty-col"
		cfg.Threshold = 0.5

		results, err := Query(context.Background(), store, embedder, "test", cfg)

		assertNoError(t, err)
		assertEmpty(t, results)
	})

	t.Run("empty results when store has no matching points", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "no-data"

		results, err := Query(context.Background(), store, embedder, "test query", cfg)

		assertNoError(t, err)
		assertEmpty(t, results)
	})

	t.Run("embedder failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "ollama down", nil)

		cfg := DefaultQueryConfig()

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		assertError(t, err)
		assertContains(t, err.Error(), "error generating query embedding")
		// Search should not be called if embedding fails
		assertEqual(t, 0, store.searchCallCount())
	})

	t.Run("search failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "qdrant timeout", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()

		_, err := Query(context.Background(), store, embedder, "test", cfg)

		assertError(t, err)
		assertContains(t, err.Error(), "error searching")
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

		assertNoError(t, err)
		assertLen(t, results, 1)

		r := results[0]
		assertEqual(t, "Full payload test.", r.Text)
		assertEqual(t, "docs/guide.md", r.Source)
		assertEqual(t, "Getting Started", r.Section)
		assertEqual(t, "documentation", r.Category)
		assertEqual(t, 5, r.ChunkIndex)
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

		assertNoError(t, err)
		assertLen(t, results, 1)
		assertEqual(t, 42, results[0].ChunkIndex)
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

		assertNoError(t, err)
		assertLen(t, results, 1)
		assertEqual(t, 7, results[0].ChunkIndex)
	})

	t.Run("results respect limit", func(t *testing.T) {
		store := newMockVectorStore()
		// Add many points
		for i := range 10 {
			store.points["test-col"] = append(store.points["test-col"], Point{
				ID:     core.Sprintf("p%d", i),
				Vector: []float32{0.1},
				Payload: map[string]any{
					"text":        core.Sprintf("Result %d", i),
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

		assertNoError(t, err)
		assertLen(t, results, 3)
	})
}

func TestQuery_Rank_Good_ChunklessSourceKeepsDistinctText(t *testing.T) {
	results := []QueryResult{
		{Text: "first chunkless hit", Source: "same.md", Score: 0.9},
		{Text: "second chunkless hit", Source: "same.md", Score: 0.8},
	}

	ranked := Rank(results, 10)

	assertLen(t, ranked, 2)
	assertEqual(t, missingChunkIndex, ranked[0].GetChunkIndex())
	assertEqual(t, "first chunkless hit", ranked[0].Text)
	assertEqual(t, "second chunkless hit", ranked[1].Text)
}
