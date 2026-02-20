package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- QueryWith tests ---

func TestQueryWith(t *testing.T) {
	t.Run("returns results from store", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["my-docs"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Hello from helper.", "source": "a.md", "section": "S", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		results, err := QueryWith(context.Background(), store, embedder, "hello", "my-docs", 5)

		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "Hello from helper.", results[0].Text)
	})

	t.Run("respects topK parameter", func(t *testing.T) {
		store := newMockVectorStore()
		for i := 0; i < 10; i++ {
			store.points["col"] = append(store.points["col"], Point{
				ID:     fmt.Sprintf("p%d", i),
				Vector: []float32{0.1},
				Payload: map[string]any{
					"text": fmt.Sprintf("Doc %d", i), "source": "d.md", "section": "", "category": "docs", "chunk_index": i,
				},
			})
		}
		embedder := newMockEmbedder(768)

		results, err := QueryWith(context.Background(), store, embedder, "test", "col", 3)

		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = fmt.Errorf("embed failed")

		_, err := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assert.Error(t, err)
	})

	t.Run("search error propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = fmt.Errorf("search failed")
		embedder := newMockEmbedder(768)

		_, err := QueryWith(context.Background(), store, embedder, "test", "col", 5)

		assert.Error(t, err)
	})
}

// --- QueryContextWith tests ---

func TestQueryContextWith(t *testing.T) {
	t.Run("returns formatted context string", func(t *testing.T) {
		store := newMockVectorStore()
		store.points["ctx-col"] = []Point{
			{ID: "1", Vector: []float32{0.1}, Payload: map[string]any{
				"text": "Context content.", "source": "guide.md", "section": "Intro", "category": "docs", "chunk_index": 0,
			}},
		}
		embedder := newMockEmbedder(768)

		result, err := QueryContextWith(context.Background(), store, embedder, "question", "ctx-col", 5)

		require.NoError(t, err)
		assert.Contains(t, result, "<retrieved_context>")
		assert.Contains(t, result, "Context content.")
		assert.Contains(t, result, "</retrieved_context>")
	})

	t.Run("empty results return empty string", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		result, err := QueryContextWith(context.Background(), store, embedder, "question", "empty", 5)

		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("error from query propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.searchErr = fmt.Errorf("broken")
		embedder := newMockEmbedder(768)

		_, err := QueryContextWith(context.Background(), store, embedder, "question", "col", 5)

		assert.Error(t, err)
	})
}

// --- IngestDirWith tests ---

func TestIngestDirWith(t *testing.T) {
	t.Run("ingests directory into collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "readme.md"), "## README\n\nProject overview.\n")
		writeFile(t, filepath.Join(dir, "guide.md"), "## Guide\n\nStep by step.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "project-docs", false)

		require.NoError(t, err)
		points := store.allPoints("project-docs")
		assert.Len(t, points, 2)
	})

	t.Run("recreate flag deletes existing collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "doc.md"), "## Doc\n\nContent.\n")

		store := newMockVectorStore()
		store.collections["col"] = 768
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, dir, "col", true)

		require.NoError(t, err)
		assert.Len(t, store.deleteCalls, 1)
		assert.Equal(t, "col", store.deleteCalls[0])
	})

	t.Run("error from ingest propagates", func(t *testing.T) {
		store := newMockVectorStore()
		store.existsErr = fmt.Errorf("exists check failed")
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp", "col", false)

		assert.Error(t, err)
	})

	t.Run("nonexistent directory returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		err := IngestDirWith(context.Background(), store, embedder, "/tmp/nonexistent-go-rag-test-dir", "col", false)

		assert.Error(t, err)
	})
}

// --- IngestFileWith tests ---

func TestIngestFileWith(t *testing.T) {
	t.Run("ingests a single file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "single.md")
		writeFile(t, path, "## Title\n\nFile content for testing.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		require.NoError(t, err)
		assert.Equal(t, 1, count)

		points := store.allPoints("col")
		require.Len(t, points, 1)
		assert.Contains(t, points[0].Payload["text"], "File content for testing.")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		_, err := IngestFileWith(context.Background(), store, embedder, "/tmp/nonexistent-test-file.md", "col")

		assert.Error(t, err)
	})

	t.Run("empty file returns zero count", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.md")
		require.NoError(t, os.WriteFile(path, []byte("  \n  "), 0644))

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("embedder error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = fmt.Errorf("embed broken")

		_, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assert.Error(t, err)
	})

	t.Run("store error propagates", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = fmt.Errorf("upsert broken")
		embedder := newMockEmbedder(768)

		_, err := IngestFileWith(context.Background(), store, embedder, path, "col")

		assert.Error(t, err)
	})
}
