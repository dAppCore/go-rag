//go:build rag

package rag

import (
	"context"
	"crypto/md5"
	"testing"
	"time"

	"dappco.re/go"
)

// --- Embedding benchmarks (Ollama on ROCm GPU) ---

func BenchmarkEmbedSingle(b *testing.B) {
	skipBenchIfOllamaUnavailable(b)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	assertNoError(b, err)

	ctx := context.Background()

	// Warm up — first call loads model into GPU memory.
	_, err = client.Embed(ctx, "warmup")
	assertNoError(b, err)

	b.ResetTimer()
	for range b.N {
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
	assertNoError(b, err)

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
	assertNoError(b, err)

	b.ResetTimer()
	for range b.N {
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
	assertNoError(b, err)

	ctx := context.Background()
	_, err = client.Embed(ctx, "warmup")
	assertNoError(b, err)

	for _, size := range []int{50, 200, 500, 1000, 2000} {
		text := repeatString("word ", size/5)
		b.Run(core.Sprintf("chars_%d", size), func(b *testing.B) {
			for range b.N {
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
	sb := core.NewBuilder()
	for i := 0; i < 50; i++ {
		sb.WriteString(core.Sprintf("## Section %d\n\n", i))
		sb.WriteString("This is a paragraph of text that represents typical documentation content. ")
		sb.WriteString("It contains technical information about software architecture and design patterns. ")
		sb.WriteString("Each section discusses different aspects of the system being documented.\n\n")
		sb.WriteString("```go\nfunc Example() error {\n\treturn nil\n}\n```\n\n")
	}
	content := sb.String()
	cfg := DefaultChunkConfig()

	b.ResetTimer()
	for range b.N {
		_ = ChunkMarkdown(content, cfg)
	}
}

func BenchmarkChunkMarkdown_VaryingSize(b *testing.B) {
	base := "This is a paragraph of text. "

	for _, paragraphs := range []int{10, 50, 200, 1000} {
		sb := core.NewBuilder()
		for i := 0; i < paragraphs; i++ {
			sb.WriteString(core.Sprintf("## Section %d\n\n", i))
			sb.WriteString(repeatString(base, 5))
			sb.WriteString("\n\n")
		}
		content := sb.String()
		cfg := DefaultChunkConfig()

		b.Run(core.Sprintf("paragraphs_%d", paragraphs), func(b *testing.B) {
			for range b.N {
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
	assertNoError(b, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	assertNoError(b, err)

	collection := "bench-search"
	dim := ollamaClient.EmbedDimension()

	// Clean up from previous runs.
	_ = qdrantClient.DeleteCollection(ctx, collection)
	err = qdrantClient.CreateCollection(ctx, collection, dim)
	assertNoError(b, err)
	defer func() { _ = qdrantClient.DeleteCollection(ctx, collection) }()

	// Seed with 100 points.
	texts := make([]string, 100)
	for i := range texts {
		texts[i] = core.Sprintf("Document %d discusses topic %d about software engineering practices and patterns.", i, i%10)
	}

	var points []Point
	for i, text := range texts {
		vec, err := ollamaClient.Embed(ctx, text)
		assertNoError(b, err)
		points = append(points, Point{
			ID:     core.Sprintf("%x", md5.Sum([]byte(core.Sprintf("bench-%d", i)))),
			Vector: vec,
			Payload: map[string]any{
				"text":     text,
				"source":   "benchmark",
				"category": core.Sprintf("topic-%d", i%10),
			},
		})
	}
	err = qdrantClient.UpsertPoints(ctx, collection, points)
	assertNoError(b, err)

	// Generate a query vector.
	queryVec, err := ollamaClient.Embed(ctx, "software engineering best practices")
	assertNoError(b, err)

	b.ResetTimer()
	for range b.N {
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
		content := core.Sprintf("# Document %d\n\nThis file covers topic %d.\n\n## Details\n\nDetailed content about software patterns and architecture decisions for component %d.\n", i, i, i)
		writeFile(b, core.JoinPath(dir, core.Sprintf("doc%d.md", i)), content)
	}

	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	assertNoError(b, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	assertNoError(b, err)

	collection := "bench-pipeline"

	b.ResetTimer()
	for range b.N {
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

func TestBenchmarkGPUEmbeddingThroughputReport(t *testing.T) {
	skipIfOllamaUnavailable(t)

	cfg := DefaultOllamaConfig()
	client, err := NewOllamaClient(cfg)
	assertNoError(t, err)

	ctx := context.Background()

	// Warm up.
	_, err = client.Embed(ctx, "warmup")
	assertNoError(t, err)

	// Single embedding latency (10 samples).
	var singleTotal time.Duration
	const singleN = 10
	for i := 0; i < singleN; i++ {
		start := time.Now()
		_, err := client.Embed(ctx, "Measure single embedding latency on ROCm GPU.")
		assertNoError(t, err)
		singleTotal += time.Since(start)
	}
	singleAvg := singleTotal / singleN

	// Batch embedding latency (10 texts, 5 samples).
	texts := make([]string, 10)
	for i := range texts {
		texts[i] = core.Sprintf("Batch text %d for throughput measurement on AMD GPU with ROCm.", i)
	}
	var batchTotal time.Duration
	const batchN = 5
	for i := 0; i < batchN; i++ {
		start := time.Now()
		_, err := client.EmbedBatch(ctx, texts)
		assertNoError(t, err)
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
func TestBenchmarkGPUSearchLatencyReport(t *testing.T) {
	skipIfQdrantUnavailable(t)
	skipIfOllamaUnavailable(t)

	ctx := context.Background()

	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	assertNoError(t, err)
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	assertNoError(t, err)

	collection := "latency-test"
	dim := ollamaClient.EmbedDimension()

	_ = qdrantClient.DeleteCollection(ctx, collection)
	err = qdrantClient.CreateCollection(ctx, collection, dim)
	assertNoError(t, err)
	defer func() { _ = qdrantClient.DeleteCollection(ctx, collection) }()

	// Seed 200 points.
	var points []Point
	for i := 0; i < 200; i++ {
		vec, err := ollamaClient.Embed(ctx, core.Sprintf("Document %d covers topic %d.", i, i%20))
		assertNoError(t, err)
		points = append(points, Point{
			ID:     core.Sprintf("%x", md5.Sum([]byte(core.Sprintf("lat-%d", i)))),
			Vector: vec,
			Payload: map[string]any{
				"text":   core.Sprintf("doc %d", i),
				"source": "latency-test",
			},
		})
	}
	err = qdrantClient.UpsertPoints(ctx, collection, points)
	assertNoError(t, err)

	queryVec, err := ollamaClient.Embed(ctx, "software engineering patterns")
	assertNoError(t, err)

	// Measure search latency (50 queries).
	var searchTotal time.Duration
	const searchN = 50
	for i := 0; i < searchN; i++ {
		start := time.Now()
		_, err := qdrantClient.Search(ctx, collection, queryVec, 5, nil)
		assertNoError(t, err)
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
