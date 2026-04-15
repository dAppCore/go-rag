package rag

import (
	"context"
	"testing"

	"dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Ingest (directory) tests with mocks ---

func TestIngest_Ingest_Good(t *testing.T) {
	t.Run("ingests markdown files from directory", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nHello world.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-col"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 1, stats.Files)
		assert.Equal(t, 1, stats.Chunks)
		assert.Equal(t, 0, stats.Errors)

		// Verify collection was created with correct dimension
		assert.Len(t, store.createCalls, 1)
		assert.Equal(t, "test-col", store.createCalls[0].Name)
		assert.Equal(t, uint64(768), store.createCalls[0].VectorSize)

		// Verify points were upserted
		points := store.allPoints("test-col")
		assert.Len(t, points, 1)
		assert.Contains(t, points[0].Payload["text"], "Hello world.")
	})

	t.Run("ingests txt files from directory", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.txt"), "## Text Section\n\nText content.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-txt"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 1, stats.Files)
		assert.Equal(t, 1, stats.Chunks)
		assert.Equal(t, 0, stats.Errors)

		points := store.allPoints("test-txt")
		require.Len(t, points, 1)
		assert.Contains(t, points[0].Payload["text"], "Text content.")
	})

	t.Run("chunks are created from input text", func(t *testing.T) {
		dir := t.TempDir()
		// Create content large enough to produce multiple chunks
		var content string
		content = "## Big Section\n\n"
		for i := range 30 {
			content += core.Sprintf("Paragraph %d with some meaningful content for testing. ", i)
			if i%3 == 0 {
				content += "\n\n"
			}
		}
		writeFile(t, core.JoinPath(dir, "large.md"), content)

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-chunks"
		cfg.Chunk = ChunkConfig{Size: 200, Overlap: 20}

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 1, stats.Files)
		assert.Greater(t, stats.Chunks, 1, "large text should produce multiple chunks")

		// Verify each chunk got an embedding call
		assert.Equal(t, stats.Chunks, embedder.embedCallCount())
	})

	t.Run("embeddings are generated for each chunk", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "a.md"), "## A\n\nContent A.\n")
		writeFile(t, core.JoinPath(dir, "b.md"), "## B\n\nContent B.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(384)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-embed"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 2, stats.Files)
		assert.Equal(t, 2, stats.Chunks)
		assert.Equal(t, 2, embedder.embedCallCount())
		assert.NotEmpty(t, embedder.batchCalls)

		// Verify vectors are the correct dimension
		points := store.allPoints("test-embed")
		for _, p := range points {
			assert.Len(t, p.Vector, 384)
		}
	})

	t.Run("points are upserted to the store", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Test\n\nSome text.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-upsert"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)
		require.NoError(t, err)

		assert.Equal(t, 1, store.upsertCallCount())
		points := store.allPoints("test-upsert")
		require.Len(t, points, 1)

		// Verify payload fields
		assert.NotEmpty(t, points[0].ID)
		assert.NotEmpty(t, points[0].Vector)
		assert.Equal(t, "doc.md", points[0].Payload["source"])
		assert.Equal(t, "Test", points[0].Payload["section"])
		assert.NotEmpty(t, points[0].Payload["category"])
		assert.Equal(t, 0, points[0].Payload["chunk_index"])
	})

	t.Run("embedder failure increments error count", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "ollama unavailable", nil)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-err"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err) // Ingest itself does not fail; it tracks errors in stats
		assert.Equal(t, 1, stats.Errors)
		assert.Equal(t, 0, stats.Chunks)

		// No points should have been upserted
		points := store.allPoints("test-err")
		assert.Empty(t, points)
	})

	t.Run("store upsert failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = core.E("mock.upsert", "qdrant connection lost", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-upsert-err"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error upserting batch")
		// Stats should still report what was processed before failure
		assert.Equal(t, 1, stats.Files)
	})

	t.Run("collection exists check failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		store.existsErr = core.E("mock.collections.exists", "connection refused", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-exists-err"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error checking collection")
	})

	t.Run("batch size handling — multiple batches", func(t *testing.T) {
		dir := t.TempDir()
		// Create enough content for multiple chunks
		for i := range 5 {
			writeFile(t, core.JoinPath(dir, core.Sprintf("doc%d.md", i)),
				core.Sprintf("## Section %d\n\nContent for document %d.\n", i, i))
		}

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-batch"
		cfg.BatchSize = 2 // Small batch size to force multiple upsert calls

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 5, stats.Files)
		assert.Equal(t, 5, stats.Chunks)

		// With 5 points and batch size 2, expect 3 upsert calls (2+2+1)
		assert.Equal(t, 3, store.upsertCallCount())

		// Verify all 5 points were stored
		points := store.allPoints("test-batch")
		assert.Len(t, points, 5)
	})

	t.Run("batch size zero defaults to 100", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-batch-zero"
		cfg.BatchSize = 0

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Equal(t, 1, stats.Chunks)
		// Should still upsert (batch size defaulted to 100)
		assert.Equal(t, 1, store.upsertCallCount())
	})

	t.Run("recreate deletes existing collection", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		// Pre-create a collection
		store.collections["test-recreate"] = 768
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-recreate"
		cfg.Recreate = true

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Len(t, store.deleteCalls, 1)
		assert.Equal(t, "test-recreate", store.deleteCalls[0])
		// Collection should be re-created after delete
		assert.Len(t, store.createCalls, 1)
	})

	t.Run("skips collection create when already exists and recreate is false", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		store.collections["existing-col"] = 768
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "existing-col"
		cfg.Recreate = false

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		assert.Empty(t, store.deleteCalls)
		assert.Empty(t, store.createCalls)
	})

	t.Run("non-existent directory returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = "/tmp/nonexistent-dir-for-go-rag-test"
		cfg.Collection = "test-nodir"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error accessing directory")
	})

	t.Run("directory with no markdown files returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "readme.go"), "package main\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-nomd"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no markdown files found")
	})

	t.Run("empty file is skipped", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "empty.md"), "   \n  \n  ")
		writeFile(t, core.JoinPath(dir, "real.md"), "## Real\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-empty"

		stats, err := Ingest(context.Background(), store, embedder, cfg, nil)

		require.NoError(t, err)
		// Only the real file should be processed
		assert.Equal(t, 1, stats.Files)
		assert.Equal(t, 1, stats.Chunks)
	})

	t.Run("progress callback is invoked", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "a.md"), "## A\n\nContent A.\n")
		writeFile(t, core.JoinPath(dir, "b.md"), "## B\n\nContent B.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-progress"

		var progressCalls []string
		progress := func(file string, chunks int, total int) {
			progressCalls = append(progressCalls, file)
		}

		_, err := Ingest(context.Background(), store, embedder, cfg, progress)

		require.NoError(t, err)
		assert.Len(t, progressCalls, 2)
	})

	t.Run("delete collection failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		store.collections["test-del-err"] = 768
		store.deleteErr = core.E("mock.collections.delete", "delete denied", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-del-err"
		cfg.Recreate = true

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error deleting collection")
	})

	t.Run("create collection failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "doc.md"), "## Section\n\nContent.\n")

		store := newMockVectorStore()
		store.createErr = core.E("mock.collections.create", "create denied", nil)
		embedder := newMockEmbedder(768)

		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-create-err"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error creating collection")
	})
}

// --- IngestFile tests with mocks ---

func TestIngest_IngestFile_Good(t *testing.T) {
	t.Run("ingests a single file", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "single.md")
		writeFile(t, path, "## Title\n\nSome content here.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		require.NoError(t, err)
		assert.Equal(t, 1, count)
		assert.Equal(t, 1, embedder.embedCallCount())

		points := store.allPoints("test-col")
		require.Len(t, points, 1)
		assert.Contains(t, points[0].Payload["text"], "Some content here.")
	})

	t.Run("empty file returns zero count", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "empty.md")
		writeFile(t, path, "   \n  ")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		require.NoError(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, 0, embedder.embedCallCount())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", "/tmp/nonexistent-file.md", DefaultChunkConfig())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error reading file")
	})

	t.Run("embedder failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed failed", nil)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error embedding chunk")
	})

	t.Run("store upsert failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = core.E("mock.upsert", "upsert failed", nil)
		embedder := newMockEmbedder(768)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error upserting points")
	})

	t.Run("payload includes correct metadata", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "docs", "architecture", "guide.md")
		writeFile(t, path, "## Architecture Guide\n\nDesign patterns and principles.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		require.NoError(t, err)
		assert.Equal(t, 1, count)

		points := store.allPoints("test-col")
		require.Len(t, points, 1)
		assert.Equal(t, "Architecture Guide", points[0].Payload["section"])
		assert.Equal(t, "architecture", points[0].Payload["category"])
		assert.Equal(t, 0, points[0].Payload["chunk_index"])
	})
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t testing.TB, path string, content string) {
	t.Helper()
	result := (&core.Fs{}).NewUnrestricted().Write(path, content)
	require.Truef(t, result.OK, "write %s: %v", path, result.Value)
}
