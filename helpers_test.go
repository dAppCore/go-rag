package rag

import (
	"context"
	"testing"

	"dappco.re/go"
)

// --- QueryWith tests ---

func TestHelpers_QueryWith_Good(t *testing.T) {
	t.Run("returns results from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["my-docs"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Hello from helper.", "source": "a.md", "section": "S", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		results, err := QueryWith(context.Background(), store, embedder, "hello", "my-docs", 5)

		assertNoError(t, err)
		assertLen(t, results, 1)
		assertEqual(t, "Hello from helper.", results[0].Text)
	})

	t.Run("respects topK parameter", func(t *testing.T) {
		store := newMockVectorStore()
		for i := range 10 {
			store.points["col"] = append(store.points["col"], Point{
				ID:     core.Sprintf("p%d", i),
				Vector: []float32{0.1},
				Payload: map[string]any{
					"text": core.Sprintf("Doc %d", i), "source": "d.md", "section": "", "category": "docs", "chunk_index": i,
				},
			})
		}
		embedder := newMockEmbedder(768)

		results, err := QueryWith(context.Background(), store, embedder, "test", "col", 3)

		assertNoError(t, err)
		assertLen(t, results, 3)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed failed", nil)

		_, err := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assertError(t, err)
	})

	t.Run("search error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "search failed", nil)
		embedder := newMockEmbedder(768)

		_, err := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assertError(t, err)
	})
}

// --- QueryContextWith tests ---

func TestHelpers_QueryContextWith_Good(t *testing.T) {
	t.Run("returns formatted context string", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["ctx-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Context content.", "source": "guide.md", "section": "Intro", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		result, err := QueryContextWith(context.Background(), store, embedder, "question", "ctx-col", 5)

		assertNoError(t, err)
		assertContains(t, result, "<retrieved_context>")
		assertContains(t, result, "Context content.")
		assertContains(t, result, "</retrieved_context>")
	})

	t.Run("empty results return empty string", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		result, err := QueryContextWith(context.Background(), store, embedder, "question", "empty", 5)

		assertNoError(t, err)
		assertEqual(t, "", result)
	})

	t.Run("error from query propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = core.E("mock.search", "broken", nil)
		embedder := newMockEmbedder(768)

		_, err := QueryContextWith(context.Background(), store, embedder, "question", "col", 5)

		assertError(t, err)
	})
}

// --- IngestDirWith tests ---

func TestHelpers_IngestDirWith_Good(t *testing.T) {
	t.Run("ingests directory into collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "readme.md"), "## README\n\nProject overview.\n")
		writeFile(t, core.JoinPath(dir, "guide.md"), "## Guide\n\nStep by step.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "project-docs", false)

		assertNoError(t, err)
		points := store.allPoints("project-docs")
		assertLen(t, points, 2)
	})

	t.Run("recreate flag deletes existing collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Doc\n\nContent.\n")

		store := newMockVectorStore()
		store.collections["col"] = 768
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "col", true)

		assertNoError(t, err)
		assertLen(t, store.deleteCalls, 1)
		assertEqual(t, "col", store.deleteCalls[0])
	})

	t.Run("error from ingest propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.existsErr = core.E("mock.collections.exists", "exists check failed", nil)
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp", "col", false)

		assertError(t, err)
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp/nonexistent-go-rag-test-dir", "col", false)

		assertError(t, err)
	})
}

// --- IngestFileWith tests ---

func TestHelpers_IngestFileWith_Good(t *testing.T) {
	t.Run("ingests a single file", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "single.md")
		writeFile(t, path, "## Title\n\nFile content for testing.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertNoError(t, err)
		assertEqual(t, 1, count)

		points := store.allPoints("col")
		assertLen(t, points, 1)
		assertContains(t, points[0].Payload["text"], "File content for testing.")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		_, err := IngestFileWith(context.Background(), store, embedder, "/tmp/nonexistent-test-file.md", "col")

		assertError(t, err)
	})

	t.Run("empty file returns zero count", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "empty.md")
		writeFile(t, path, "  \n  ")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertNoError(t, err)
		assertEqual(t, 0, count)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed broken", nil)

		_, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertError(t, err)
	})

	t.Run("store error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = core.E("mock.upsert", "upsert broken", nil)
		embedder := newMockEmbedder(768)

		_, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assertError(t, err)
	})
}

// --- JoinResults tests ---

func TestHelpers_JoinResults_Good(t *testing.T) {
	t.Run("joins non-empty result text with blank lines", func(t *testing.T) {
		results := []QueryResult{
			{Text: "First result."},
			{Text: "   "},
			{Text: "Second result."},
		}

		output := JoinResults(results)

		assertEqual(t, "First result.\n\nSecond result.", output)
	})

	t.Run("works with SearchResult values", func(t *testing.T) {
		results := []SearchResult{
			{Payload: map[string]any{"text": "Alpha."}},
			{},
			{Payload: map[string]any{"text": "Beta."}},
		}

		output := JoinResults(results)

		assertEqual(t, "Alpha.\n\nBeta.", output)
	})

	t.Run("empty input returns empty string", func(t *testing.T) {
		assertEqual(t, "", JoinResults[QueryResult](nil))
		assertEqual(t, "", JoinResults([]QueryResult{}))
	})
}
