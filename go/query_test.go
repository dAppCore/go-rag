package rag

import (
	"context"
	"iter"
	"testing"

	"dappco.re/go"
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

		r := Query(context.Background(), store, embedder, "what is Go?", cfg)

		assertNoError(t, r)
		assertEqual(t, 1, embedder.embedCallCount())
		assertEqual(t, "what is Go?", embedder.embedCalls[0])
	})

	t.Run("search is called with correct parameters", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "my-docs"
		cfg.Limit = 3

		r := Query(context.Background(), store, embedder, "test query", cfg)

		assertNoError(t, r)
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

		r := Query(context.Background(), store, embedder, "test", cfg)

		assertNoError(t, r)
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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

		assertEmpty(t, results)
	})

	t.Run("empty results when store has no matching points", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()
		cfg.Collection = "no-data"

		r := Query(context.Background(), store, embedder, "test query", cfg)
		results := resultValue[[]QueryResult](t, r)

		assertEmpty(t, results)
	})

	t.Run("embedder failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "ollama down", nil)

		cfg := DefaultQueryConfig()

		r := Query(context.Background(), store, embedder, "test", cfg)

		assertError(t, r)
		assertContains(t, r.Error(), "error generating query embedding")
		// Search should not be called if embedding fails
		assertEqual(t, 0, store.searchCallCount())
	})

	t.Run("search failure returns error", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "qdrant timeout", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultQueryConfig()

		r := Query(context.Background(), store, embedder, "test", cfg)

		assertError(t, r)
		assertContains(t, r.Error(), "error searching")
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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

		assertLen(t, results, 1)

		result := results[0]
		assertEqual(t, "Full payload test.", result.Text)
		assertEqual(t, "docs/guide.md", result.Source)
		assertEqual(t, "Getting Started", result.Section)
		assertEqual(t, "documentation", result.Category)
		assertEqual(t, 5, result.ChunkIndex)
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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

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

		r := Query(context.Background(), store, embedder, "test", cfg)
		results := resultValue[[]QueryResult](t, r)

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

func TestQuery_QueryResult_GetText_Good(t *core.T) {
	result := QueryResult{Text: "answer text"}

	core.AssertEqual(t, "answer text", result.GetText())
	core.AssertNotEmpty(t, result.GetText())
}

func TestQuery_QueryResult_GetText_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, "", result.GetText())
	core.AssertEmpty(t, result.GetText())
}

func TestQuery_QueryResult_GetText_Ugly(t *core.T) {
	result := QueryResult{Text: "<xml>&text"}

	core.AssertContains(t, result.GetText(), "&")
	core.AssertEqual(t, "<xml>&text", result.GetText())
}

func TestQuery_QueryResult_GetScore_Good(t *core.T) {
	result := QueryResult{Score: 0.8}

	core.AssertEqual(t, float32(0.8), result.GetScore())
	core.AssertGreater(t, result.GetScore(), float32(0))
}

func TestQuery_QueryResult_GetScore_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, float32(0), result.GetScore())
	core.AssertFalse(t, result.GetScore() > 0)
}

func TestQuery_QueryResult_GetScore_Ugly(t *core.T) {
	result := QueryResult{Score: -0.2}

	core.AssertEqual(t, float32(-0.2), result.GetScore())
	core.AssertLess(t, result.GetScore(), float32(0))
}

func TestQuery_QueryResult_GetSource_Good(t *core.T) {
	result := QueryResult{Source: "docs/source.md"}

	core.AssertEqual(t, "docs/source.md", result.GetSource())
	core.AssertContains(t, result.GetSource(), "source")
}

func TestQuery_QueryResult_GetSource_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, "", result.GetSource())
	core.AssertEmpty(t, result.GetSource())
}

func TestQuery_QueryResult_GetSource_Ugly(t *core.T) {
	result := QueryResult{Source: "docs/source with spaces.md"}

	core.AssertContains(t, result.GetSource(), "spaces")
	core.AssertEqual(t, "docs/source with spaces.md", result.GetSource())
}

func TestQuery_QueryResult_HasChunkIndex_Good(t *core.T) {
	result := QueryResult{ChunkIndex: 0, ChunkIndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 0, result.GetChunkIndex())
}

func TestQuery_QueryResult_HasChunkIndex_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertFalse(t, result.HasChunkIndex())
	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
}

func TestQuery_QueryResult_HasChunkIndex_Ugly(t *core.T) {
	result := QueryResult{Index: 9, IndexPresent: true}

	core.AssertTrue(t, result.HasChunkIndex())
	core.AssertEqual(t, 9, result.GetChunkIndex())
}

func TestQuery_QueryResult_GetChunkIndex_Good(t *core.T) {
	result := QueryResult{ChunkIndex: 5, ChunkIndexPresent: true}

	core.AssertEqual(t, 5, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestQuery_QueryResult_GetChunkIndex_Bad(t *core.T) {
	result := QueryResult{}

	core.AssertEqual(t, missingChunkIndex, result.GetChunkIndex())
	core.AssertFalse(t, result.HasChunkIndex())
}

func TestQuery_QueryResult_GetChunkIndex_Ugly(t *core.T) {
	result := QueryResult{Index: 0, IndexPresent: true}

	core.AssertEqual(t, 0, result.GetChunkIndex())
	core.AssertTrue(t, result.HasChunkIndex())
}

func TestQuery_DefaultQueryConfig_Bad(t *core.T) {
	cfg := DefaultQueryConfig()

	core.AssertNotEqual(t, "", cfg.Collection)
	core.AssertNotEqual(t, uint64(0), cfg.Limit)
}

func TestQuery_DefaultQueryConfig_Ugly(t *core.T) {
	cfg := DefaultQueryConfig()
	cfg.Collection = "mutated"

	core.AssertEqual(t, "hostuk-docs", DefaultQueryConfig().Collection)
	core.AssertEqual(t, "mutated", cfg.Collection)
}

func TestQuery_Rank_Good(t *core.T) {
	results := []QueryResult{{Text: "low", Score: 0.1}, {Text: "high", Score: 0.9}}
	ranked := Rank(results, 1)

	core.AssertLen(t, ranked, 1)
	core.AssertEqual(t, "high", ranked[0].Text)
}

func TestQuery_Rank_Bad(t *core.T) {
	ranked := Rank([]QueryResult{{Text: "ignored", Score: 1}}, 0)

	core.AssertEmpty(t, ranked)
	core.AssertEqual(t, 0, len(ranked))
}

func TestQuery_Rank_Ugly(t *core.T) {
	results := []QueryResult{{Text: "dup", Source: "a.md", ChunkIndex: 1, ChunkIndexPresent: true, Score: 0.9}, {Text: "dup", Source: "a.md", ChunkIndex: 1, ChunkIndexPresent: true, Score: 0.8}}
	ranked := Rank(results, 5)

	core.AssertLen(t, ranked, 1)
	core.AssertEqual(t, float32(0.9), ranked[0].Score)
}

func TestQuery_Query_Bad(t *core.T) {
	store := newMockVectorStore()
	store.searchErr = core.NewError("search failed")
	r := Query(core.Background(), store, newMockEmbedder(2), "query", DefaultQueryConfig())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "error searching")
}

func TestQuery_Query_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 0.1, Payload: map[string]any{"text": "low"}}}, nil
	}
	r := Query(core.Background(), store, newMockEmbedder(2), "query", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.9})
	results := r.Value.([]QueryResult)

	core.AssertTrue(t, r.OK)
	core.AssertEmpty(t, results)
}

func TestQuery_QuerySeq_Good(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 0.9, Payload: map[string]any{"text": "hit", "source": "a.md", "chunk_index": 0}}}, nil
	}
	r := QuerySeq(core.Background(), store, newMockEmbedder(2), "query", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.1})
	seq := r.Value.(iter.Seq[QueryResult])

	var results []QueryResult
	for result := range seq {
		results = append(results, result)
	}
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, results, 1)
}

func TestQuery_QuerySeq_Bad(t *core.T) {
	embedder := newMockEmbedder(2)
	embedder.embedErr = core.NewError("embed failed")
	r := QuerySeq(core.Background(), newMockVectorStore(), embedder, "query", DefaultQueryConfig())

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "embedding")
}

func TestQuery_QuerySeq_Ugly(t *core.T) {
	store := newMockVectorStore()
	store.searchFunc = func(string, []float32, uint64, map[string]string) ([]SearchResult, error) {
		return []SearchResult{{Score: 1, Payload: map[string]any{"text": "Kubernetes"}}, {Score: 0.95, Payload: map[string]any{"text": "Other"}}}, nil
	}
	r := QuerySeq(core.Background(), store, newMockEmbedder(2), "kubernetes", QueryConfig{Collection: "docs", Limit: 5, Threshold: 0.1, Keywords: true})
	seq := r.Value.(iter.Seq[QueryResult])

	var results []QueryResult
	for result := range seq {
		results = append(results, result)
	}
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "Kubernetes", results[0].Text)
}

func TestQuery_FormatResultsText_Bad(t *core.T) {
	text := FormatResultsText(nil)

	core.AssertEqual(t, "No results found.", text)
	core.AssertNotContains(t, text, "Result 1")
}

func TestQuery_FormatResultsText_Ugly(t *core.T) {
	text := FormatResultsText([]QueryResult{{Text: "", Source: "", Category: "", Score: 0}})

	core.AssertContains(t, text, "score: 0.00")
	core.AssertContains(t, text, "Category:")
}

func TestQuery_FormatResultsContext_Bad(t *core.T) {
	context := FormatResultsContext(nil)

	core.AssertEqual(t, "", context)
	core.AssertEmpty(t, context)
}

func TestQuery_FormatResultsContext_Ugly(t *core.T) {
	context := FormatResultsContext([]QueryResult{{Text: "<tag>&", Source: "a&b.md", Section: "\"sec\""}})

	core.AssertContains(t, context, "&lt;tag&gt;&amp;")
	core.AssertContains(t, context, "a&amp;b.md")
}

func TestQuery_FormatResultsJSON_Bad(t *core.T) {
	json := FormatResultsJSON(nil)

	core.AssertEqual(t, "[]", json)
	core.AssertLen(t, json, 2)
}

func TestQuery_FormatResultsJSON_Ugly(t *core.T) {
	json := FormatResultsJSON([]QueryResult{{Text: "line\nquote\"", Source: "a.md", Score: 0.123456}})

	core.AssertContains(t, json, "0.1235")
	core.AssertContains(t, json, `quote\"`)
}
