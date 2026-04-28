package rag

import (
	"context"
	"testing"

	"dappco.re/go"
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

		assertNoError(t, err)
		assertEqual(t, 1, stats.Files)
		assertEqual(t, 1, stats.Chunks)
		assertEqual(t, 0, stats.Errors)

		// Verify collection was created with correct dimension
		assertLen(t, store.createCalls, 1)
		assertEqual(t, "test-col", store.createCalls[0].Name)
		assertEqual(t, uint64(768), store.createCalls[0].VectorSize)

		// Verify points were upserted
		points := store.allPoints("test-col")
		assertLen(t, points, 1)
		assertContains(t, points[0].Payload["text"], "Hello world.")
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

		assertNoError(t, err)
		assertEqual(t, 1, stats.Files)
		assertEqual(t, 1, stats.Chunks)
		assertEqual(t, 0, stats.Errors)

		points := store.allPoints("test-txt")
		assertLen(t, points, 1)
		assertContains(t, points[0].Payload["text"], "Text content.")
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

		assertNoError(t, err)
		assertEqual(t, 1, stats.Files)
		assertGreater(t, stats.Chunks, 1, "large text should produce multiple chunks")

		// Verify each chunk got an embedding call
		assertEqual(t, stats.Chunks, embedder.embedCallCount())
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

		assertNoError(t, err)
		assertEqual(t, 2, stats.Files)
		assertEqual(t, 2, stats.Chunks)
		assertEqual(t, 2, embedder.embedCallCount())
		assertLen(t, embedder.batchCalls, 2)

		// Verify vectors are the correct dimension
		points := store.allPoints("test-embed")
		for _, p := range points {
			assertLen(t, p.Vector, 384)
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
		assertNoError(t, err)

		assertEqual(t, 1, store.upsertCallCount())
		points := store.allPoints("test-upsert")
		assertLen(t, points, 1)

		// Verify payload fields
		assertNotEmpty(t, points[0].ID)
		assertNotEmpty(t, points[0].Vector)
		assertEqual(t, "doc.md", points[0].Payload["source"])
		assertEqual(t, "Test", points[0].Payload["section"])
		assertNotEmpty(t, points[0].Payload["category"])
		assertEqual(t, 0, points[0].Payload["chunk_index"])
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

		assertNoError(t, err) // Ingest itself does not fail; it tracks errors in stats
		assertEqual(t, 1, stats.Errors)
		assertEqual(t, 0, stats.Chunks)

		// No points should have been upserted
		points := store.allPoints("test-err")
		assertEmpty(t, points)
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

		assertError(t, err)
		assertContains(t, err.Error(), "error upserting batch")
		// Stats should still report what was processed before failure
		assertEqual(t, 1, stats.Files)
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

		assertError(t, err)
		assertContains(t, err.Error(), "error checking collection")
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

		assertNoError(t, err)
		assertEqual(t, 5, stats.Files)
		assertEqual(t, 5, stats.Chunks)

		// With 5 points and batch size 2, expect 3 upsert calls (2+2+1)
		assertEqual(t, 3, store.upsertCallCount())

		// Verify all 5 points were stored
		points := store.allPoints("test-batch")
		assertLen(t, points, 5)
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

		assertNoError(t, err)
		assertEqual(t, 1, stats.Chunks)
		// Should still upsert (batch size defaulted to 100)
		assertEqual(t, 1, store.upsertCallCount())
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

		assertNoError(t, err)
		assertLen(t, store.deleteCalls, 1)
		assertEqual(t, "test-recreate", store.deleteCalls[0])
		// Collection should be re-created after delete
		assertLen(t, store.createCalls, 1)
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

		assertNoError(t, err)
		assertEmpty(t, store.deleteCalls)
		assertEmpty(t, store.createCalls)
	})

	t.Run("non-existent directory returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = "/tmp/nonexistent-dir-for-go-rag-test"
		cfg.Collection = "test-nodir"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assertError(t, err)
		assertContains(t, err.Error(), "error accessing directory")
	})

	t.Run("directory with no matching files returns error", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, core.JoinPath(dir, "readme.go"), "package main\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "test-nomd"

		_, err := Ingest(context.Background(), store, embedder, cfg, nil)

		assertError(t, err)
		assertContains(t, err.Error(), "no matching files found")
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

		assertNoError(t, err)
		// Only the real file should be processed
		assertEqual(t, 1, stats.Files)
		assertEqual(t, 1, stats.Chunks)
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

		assertNoError(t, err)
		assertLen(t, progressCalls, 2)
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

		assertError(t, err)
		assertContains(t, err.Error(), "error deleting collection")
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

		assertError(t, err)
		assertContains(t, err.Error(), "error creating collection")
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

		assertNoError(t, err)
		assertEqual(t, 1, count)
		assertEqual(t, 1, embedder.embedCallCount())

		points := store.allPoints("test-col")
		assertLen(t, points, 1)
		assertContains(t, points[0].Payload["text"], "Some content here.")
	})

	t.Run("empty file returns zero count", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "empty.md")
		writeFile(t, path, "   \n  ")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assertNoError(t, err)
		assertEqual(t, 0, count)
		assertEqual(t, 0, embedder.embedCallCount())
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", "/tmp/nonexistent-file.md", DefaultChunkConfig())

		assertError(t, err)
		assertContains(t, err.Error(), "error reading file")
	})

	t.Run("embedder failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		embedder.embedErr = core.E("mock.embed", "embed failed", nil)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assertError(t, err)
		assertContains(t, err.Error(), "error embedding chunk")
	})

	t.Run("store upsert failure returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.md")
		writeFile(t, path, "## Title\n\nContent.\n")

		store := newMockVectorStore()
		store.upsertErr = core.E("mock.upsert", "upsert failed", nil)
		embedder := newMockEmbedder(768)

		_, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assertError(t, err)
		assertContains(t, err.Error(), "error upserting points")
	})

	t.Run("pdf fallback reads plaintext files with pdf extension", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "doc.pdf")
		writeFile(t, path, "## Title\n\nPlaintext content in a mislabeled pdf file.\n")

		content, err := readDocument((&core.Fs{}).NewUnrestricted(), path)

		assertNoError(t, err)
		assertContains(t, content, "Plaintext content")
	})

	t.Run("payload includes correct metadata", func(t *testing.T) {
		dir := t.TempDir()
		path := core.JoinPath(dir, "docs", "architecture", "guide.md")
		writeFile(t, path, "## Architecture Guide\n\nDesign patterns and principles.\n")

		store := newMockVectorStore()
		embedder := newMockEmbedder(768)

		count, err := IngestFile(context.Background(), store, embedder, "test-col", path, DefaultChunkConfig())

		assertNoError(t, err)
		assertEqual(t, 1, count)

		points := store.allPoints("test-col")
		assertLen(t, points, 1)
		assertEqual(t, "Architecture Guide", points[0].Payload["section"])
		assertEqual(t, "architecture", points[0].Payload["category"])
		assertEqual(t, 0, points[0].Payload["chunk_index"])
	})
}

// writeFile is a test helper that creates a file with the given content.
func writeFile(t testing.TB, path string, content string) {
	t.Helper()
	result := (&core.Fs{}).NewUnrestricted().Write(path, content)
	assertTruef(t, result.OK, "write %s: %v", path, result.Value)
}
