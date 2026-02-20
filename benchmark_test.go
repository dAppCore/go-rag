package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// generateMarkdownDoc creates a ~10KB markdown document for benchmarking.
func generateMarkdownDoc() string {
	var sb strings.Builder
	sb.WriteString("# Benchmark Document\n\n")
	sb.WriteString("This document is generated for benchmarking the chunking pipeline.\n\n")

	for i := 0; i < 20; i++ {
		sb.WriteString(fmt.Sprintf("## Section %d\n\n", i+1))
		for j := 0; j < 5; j++ {
			sb.WriteString(fmt.Sprintf(
				"Paragraph %d in section %d contains representative text for testing. "+
					"It includes multiple sentences to exercise the sentence-aware splitter. "+
					"The content is deliberately verbose to reach a realistic document size. "+
					"Each paragraph is unique to avoid deduplication effects.\n\n",
				j+1, i+1))
		}
	}

	return sb.String()
}

// generateQueryResults creates n QueryResult entries for benchmarking.
func generateQueryResults(n int) []QueryResult {
	results := make([]QueryResult, n)
	for i := range results {
		results[i] = QueryResult{
			Text:       fmt.Sprintf("This is result number %d with some representative text content for testing purposes.", i+1),
			Source:     fmt.Sprintf("docs/section-%d/file-%d.md", i/5, i),
			Section:    fmt.Sprintf("Section %d", i+1),
			Category:   "documentation",
			ChunkIndex: i,
			Score:      1.0 - float32(i)*0.01,
		}
	}
	return results
}

// --- BenchmarkChunk ---

func BenchmarkChunk(b *testing.B) {
	doc := generateMarkdownDoc()
	cfg := DefaultChunkConfig()

	b.ResetTimer()
	for b.Loop() {
		ChunkMarkdown(doc, cfg)
	}
}

// --- BenchmarkChunkWithOverlap ---

func BenchmarkChunkWithOverlap(b *testing.B) {
	doc := generateMarkdownDoc()
	cfg := ChunkConfig{Size: 500, Overlap: 100}

	b.ResetTimer()
	for b.Loop() {
		ChunkMarkdown(doc, cfg)
	}
}

// --- BenchmarkQuery_Mock ---

func BenchmarkQuery_Mock(b *testing.B) {
	store := newMockVectorStore()
	store.collections["bench-col"] = 768
	// Pre-populate with 50 points
	for i := 0; i < 50; i++ {
		store.points["bench-col"] = append(store.points["bench-col"], Point{
			ID:     fmt.Sprintf("p%d", i),
			Vector: make([]float32, 768),
			Payload: map[string]any{
				"text":        fmt.Sprintf("Benchmark document %d with relevant content.", i),
				"source":      fmt.Sprintf("doc%d.md", i),
				"section":     "Section",
				"category":    "docs",
				"chunk_index": i,
			},
		})
	}
	embedder := newMockEmbedder(768)
	cfg := DefaultQueryConfig()
	cfg.Collection = "bench-col"
	cfg.Threshold = 0.0
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = Query(ctx, store, embedder, "benchmark query text", cfg)
	}
}

// --- BenchmarkIngest_Mock ---

func BenchmarkIngest_Mock(b *testing.B) {
	dir := b.TempDir()
	// Create 10 markdown files
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("## File %d\n\nThis is file number %d with some test content for benchmarking.\n", i, i)
		path := filepath.Join(dir, fmt.Sprintf("doc%d.md", i))
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			b.Fatal(err)
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "bench-ingest"

		_, _ = Ingest(ctx, store, embedder, cfg, nil)
	}
}

// --- BenchmarkFormatResults ---

func BenchmarkFormatResultsText(b *testing.B) {
	results := generateQueryResults(20)

	b.ResetTimer()
	for b.Loop() {
		FormatResultsText(results)
	}
}

func BenchmarkFormatResultsContext(b *testing.B) {
	results := generateQueryResults(20)

	b.ResetTimer()
	for b.Loop() {
		FormatResultsContext(results)
	}
}

func BenchmarkFormatResultsJSON(b *testing.B) {
	results := generateQueryResults(20)

	b.ResetTimer()
	for b.Loop() {
		FormatResultsJSON(results)
	}
}

// --- BenchmarkKeywordFilter ---

func BenchmarkKeywordFilter(b *testing.B) {
	results := generateQueryResults(100)
	keywords := []string{"result", "content", "testing", "documentation", "section"}

	b.ResetTimer()
	for b.Loop() {
		KeywordFilter(results, keywords)
	}
}
