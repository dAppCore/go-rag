//go:build rag

package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// --- Embedding benchmarks (Ollama on ROCm GPU) ---

func BenchmarkEmbedSingle(b *testing.B) {
	skipBenchIfOllamaUnavailable(b)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	require.NoError(b, err)

	ctx := context.Background()

	// Warm up — first call loads model into GPU memory.
	_, err = client.Embed(ctx, "warmup")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Embed(ctx, "The quick brown fox jumps over the lazy dog.")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmbedBatch(b *testing.B) {
	skipBenchIfOllamaUnavailable(b)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	require.NoError(b, err)

	ctx := context.Background()

	texts := []string{
		"Go is a statically typed programming language designed at Google.",
		"Rust prioritises memory safety without a garbage collector.",
		"Python is widely used for data science and machine learning.",
		"TypeScript adds static types to JavaScript for better tooling.",
		"Zig is a systems programming language with manual memory management.",
		"Elixir runs on the BEAM VM for fault-tolerant distributed systems.",
		"Haskell is a purely functional programming language with lazy evaluation.",
		"C++ remains dominant in game engines and high-performance computing.",
		"Ruby emphasises developer happiness with elegant syntax.",
		"Kotlin is the preferred language for Android development.",
	}

	// Warm up.
	_, err = client.Embed(ctx, "warmup")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.EmbedBatch(ctx, texts)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEmbedVaryingLength measures embedding latency across text lengths.
func BenchmarkEmbedVaryingLength(b *testing.B) {
	skipBenchIfOllamaUnavailable(b)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	require.NoError(b, err)

	ctx := context.Background()
	_, err = client.Embed(ctx, "warmup")
	require.NoError(b, err)

	for _, size := range []int{50, 200, 500, 1000, 2000} {
		text := strings.Repeat("word ", size/5)
		b.Run(fmt.Sprintf("chars_%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := client.Embed(ctx, text)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// --- Chunking benchmarks (pure CPU, varying sizes) ---

func BenchmarkChunkMarkdown_GPU(b *testing.B) {
	// Generate a realistic markdown document.
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString(fmt.Sprintf("## Section %d\n\n", i))
		sb.WriteString("This is a paragraph of text that represents typical documentation content. ")
		sb.WriteString("It contains technical information about software architecture and design patterns. ")
		sb.WriteString("Each section discusses different aspects of the system being documented.\n\n")
		sb.WriteString("```go\nfunc Example() error {\n\treturn nil\n}\n```\n\n")
	}
	content := sb.String()
	cfg := DefaultChunkConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ChunkMarkdown(content, cfg)
	}
}

func BenchmarkChunkMarkdown_VaryingSize(b *testing.B) {
	base := "This is a paragraph of text. "

	for _, paragraphs := range []int{10, 50, 200, 1000} {
		var sb strings.Builder
		for i := 0; i < paragraphs; i++ {
			sb.WriteString(fmt.Sprintf("## Section %d\n\n", i))
			sb.WriteString(strings.Repeat(base, 5))
			sb.WriteString("\n\n")
		}
		content := sb.String()
		cfg := DefaultChunkConfig()

		b.Run(fmt.Sprintf("paragraphs_%d", paragraphs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = ChunkMarkdown(content, cfg)
			}
		})
	}
}

// --- Search latency benchmarks (Qdrant) ---

func BenchmarkQdrantSearch(b *testing.B) {
	skipBenchIfQdrantUnavailable(b)
	skipBenchIfOllamaUnavailable(b)

	ctx := context.Background()

	// Set up Qdrant with test data.
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	require.NoError(b, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	require.NoError(b, err)

	collection := "bench-search"
	dim := ollamaClient.EmbedDimension()

	// Clean up from previous runs.
	_ = qdrantClient.DeleteCollection(ctx, collection)
	err = qdrantClient.CreateCollection(ctx, collection, dim)
	require.NoError(b, err)
	defer func() { _ = qdrantClient.DeleteCollection(ctx, collection) }()

	// Seed with 100 points.
	texts := make([]string, 100)
	for i := range texts {
		texts[i] = fmt.Sprintf("Document %d discusses topic %d about software engineering practices and patterns.", i, i%10)
	}

	var points []Point
	for i, text := range texts {
		vec, err := ollamaClient.Embed(ctx, text)
		require.NoError(b, err)
		points = append(points, Point{
			ID:     fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("bench-%d", i)))),
			Vector: vec,
			Payload: map[string]any{
				"text":     text,
				"source":   "benchmark",
				"category": fmt.Sprintf("topic-%d", i%10),
			},
		})
	}
	err = qdrantClient.UpsertPoints(ctx, collection, points)
	require.NoError(b, err)

	// Generate a query vector.
	queryVec, err := ollamaClient.Embed(ctx, "software engineering best practices")
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := qdrantClient.Search(ctx, collection, queryVec, 5, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// --- Full pipeline benchmark (ingest + query) ---

func BenchmarkFullPipeline(b *testing.B) {
	skipBenchIfQdrantUnavailable(b)
	skipBenchIfOllamaUnavailable(b)

	ctx := context.Background()

	// Create temp dir with markdown files.
	dir := b.TempDir()
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("# Document %d\n\nThis file covers topic %d.\n\n## Details\n\nDetailed content about software patterns and architecture decisions for component %d.\n", i, i, i)
		err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("doc%d.md", i)), []byte(content), 0644)
		require.NoError(b, err)
	}

	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	require.NoError(b, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	require.NoError(b, err)

	collection := "bench-pipeline"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Ingest
		cfg := DefaultIngestConfig()
		cfg.Directory = dir
		cfg.Collection = collection
		cfg.Recreate = true
		_, err := Ingest(ctx, qdrantClient, ollamaClient, cfg, nil)
		if err != nil {
			b.Fatal(err)
		}

		// Query
		_, err = Query(ctx, qdrantClient, ollamaClient, "software architecture", QueryConfig{
			Collection: collection,
			Limit:      3,
			Threshold:  0.0,
		})
		if err != nil {
			b.Fatal(err)
		}
	}

	// Clean up.
	_ = qdrantClient.DeleteCollection(ctx, collection)
}

// --- Embedding throughput test (not a benchmark — reports human-readable stats) ---

func TestEmbeddingThroughput(t *testing.T) {
	skipIfOllamaUnavailable(t)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Warm up.
	_, err = client.Embed(ctx, "warmup")
	require.NoError(t, err)

	// Single embedding latency (10 samples).
	var singleTotal time.Duration
	const singleN = 10
	for i := 0; i < singleN; i++ {
		start := time.Now()
		_, err := client.Embed(ctx, "Measure single embedding latency on ROCm GPU.")
		require.NoError(t, err)
		singleTotal += time.Since(start)
	}
	singleAvg := singleTotal / singleN

	// Batch embedding latency (10 texts, 5 samples).
	texts := make([]string, 10)
	for i := range texts {
		texts[i] = fmt.Sprintf("Batch text %d for throughput measurement on AMD GPU with ROCm.", i)
	}
	var batchTotal time.Duration
	const batchN = 5
	for i := 0; i < batchN; i++ {
		start := time.Now()
		_, err := client.EmbedBatch(ctx, texts)
		require.NoError(t, err)
		batchTotal += time.Since(start)
	}
	batchAvg := batchTotal / batchN

	t.Logf("--- Embedding Throughput (nomic-embed-text, ROCm GPU) ---")
	t.Logf("Single embed:  %v avg (%d samples)", singleAvg, singleN)
	t.Logf("Batch (10):    %v avg (%d samples)", batchAvg, batchN)
	t.Logf("Per-text in batch: %v", batchAvg/10)
	t.Logf("Throughput:    %.1f embeds/sec (single), %.1f embeds/sec (batch)",
		float64(time.Second)/float64(singleAvg),
		float64(time.Second)/float64(batchAvg)*10)
}

// TestSearchLatency reports Qdrant search timing.
func TestSearchLatency(t *testing.T) {
	skipIfQdrantUnavailable(t)
	skipIfOllamaUnavailable(t)

	ctx := context.Background()

	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	require.NoError(t, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	require.NoError(t, err)

	collection := "latency-test"
	dim := ollamaClient.EmbedDimension()

	_ = qdrantClient.DeleteCollection(ctx, collection)
	err = qdrantClient.CreateCollection(ctx, collection, dim)
	require.NoError(t, err)
	defer func() { _ = qdrantClient.DeleteCollection(ctx, collection) }()

	// Seed 200 points.
	var points []Point
	for i := 0; i < 200; i++ {
		vec, err := ollamaClient.Embed(ctx, fmt.Sprintf("Document %d covers topic %d.", i, i%20))
		require.NoError(t, err)
		points = append(points, Point{
			ID:     fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("lat-%d", i)))),
			Vector: vec,
			Payload: map[string]any{
				"text":   fmt.Sprintf("doc %d", i),
				"source": "latency-test",
			},
		})
	}
	err = qdrantClient.UpsertPoints(ctx, collection, points)
	require.NoError(t, err)

	queryVec, err := ollamaClient.Embed(ctx, "software engineering patterns")
	require.NoError(t, err)

	// Measure search latency (50 queries).
	var searchTotal time.Duration
	const searchN = 50
	for i := 0; i < searchN; i++ {
		start := time.Now()
		_, err := qdrantClient.Search(ctx, collection, queryVec, 5, nil)
		require.NoError(t, err)
		searchTotal += time.Since(start)
	}
	searchAvg := searchTotal / searchN

	t.Logf("--- Search Latency (200 points, top-5) ---")
	t.Logf("Avg: %v (%d queries)", searchAvg, searchN)
	t.Logf("QPS: %.0f queries/sec", float64(time.Second)/float64(searchAvg))
}

// --- Helpers ---

func skipBenchIfOllamaUnavailable(b *testing.B) {
	b.Helper()
	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	if err != nil {
		b.Skip("Ollama not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.VerifyModel(ctx); err != nil {
		b.Skip("Ollama model not available")
	}
}

func skipBenchIfQdrantUnavailable(b *testing.B) {
	b.Helper()
	cfg := DefaultQdrantConfig()
	client, err := NewQdrantClient(cfg)
	if err != nil {
		b.Skip("Qdrant not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.HealthCheck(ctx); err != nil {
		b.Skip("Qdrant health check failed")
	}
	_ = client.Close()
}
