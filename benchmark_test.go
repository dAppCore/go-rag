package rag

import (
	"context"
	"testing"

	"dappco.re/go"
)

// generateMarkdownDoc creates a ~10KB markdown document for benchmarking.
func generateMarkdownDoc() string {
	sb := core.NewBuilder()
	sb.WriteString("# Benchmark Document\n\n")
	sb.WriteString("This document is generated for benchmarking the chunking pipeline.\n\n")

	for i := range 20 {
		sb.WriteString(core.Sprintf("## Section %d\n\n", i+1))
		for j := range 5 {
			sb.WriteString(core.Sprintf(
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
			Text:       core.Sprintf("This is result number %d with some representative text content for testing purposes.", i+1),
			Source:     core.Sprintf("docs/section-%d/file-%d.md", i/5, i),
			Section:    core.Sprintf("Section %d", i+1),
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
	for i := range 50 {
		store.points["bench-col"] = append(store.points["bench-col"], Point{
			ID:     core.Sprintf("p%d", i),
			Vector: make([]float32, 768),
			Payload: map[string]any{
				"text":        core.Sprintf("Benchmark document %d with relevant content.", i),
				"source":      core.Sprintf("doc%d.md", i),
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
		_ = Query(ctx, store, embedder, "benchmark query text", cfg)
	}
}

// --- BenchmarkIngest_Mock ---

func BenchmarkIngest_Mock(b *testing.B) {
	dir := b.TempDir()
	// Create 10 markdown files
	for i := range 10 {
		content := core.Sprintf("## File %d\n\nThis is file number %d with some test content for benchmarking.\n", i, i)
		path := core.JoinPath(dir, core.Sprintf("doc%d.md", i))
		writeFile(b, path, content)
	}

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		store := newMockVectorStore()
		embedder := newMockEmbedder(768)
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = "bench-ingest"

		_ = Ingest(ctx, store, embedder, cfg, nil)
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
