---
title: Architecture
description: Internal design of go-rag — core interfaces, chunking strategy, ingestion and query pipelines, keyword boosting, result formatting, and performance characteristics.
---

# Architecture

This document explains how `forge.lthn.ai/core/go-rag` works internally. It covers the core interfaces that define the abstraction boundaries, the three-level Markdown chunking strategy, the ingestion and query pipelines, keyword boosting, result formatting, and measured performance characteristics.

## Core Interfaces

All business logic operates against two interfaces. Concrete clients (`OllamaClient`, `QdrantClient`) satisfy them, and test doubles (`mockEmbedder`, `mockVectorStore`) enable full mock-based testing without external services.

### Embedder

Defined in `embedder.go`:

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    EmbedDimension() uint64
}
```

`OllamaClient` is the production implementation. The interface enables swapping in any embedding provider (OpenAI, Cohere, a local model) without changing pipeline code.

### VectorStore

Defined in `vectorstore.go`:

```go
type VectorStore interface {
    CreateCollection(ctx context.Context, name string, vectorSize uint64) error
    CollectionExists(ctx context.Context, name string) (bool, error)
    DeleteCollection(ctx context.Context, name string) error
    ListCollections(ctx context.Context) ([]string, error)
    CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error)
    UpsertPoints(ctx context.Context, collection string, points []Point) error
    Search(ctx context.Context, collection string, vector []float32, limit uint64,
        filter map[string]string) ([]SearchResult, error)
}
```

`QdrantClient` is the production implementation. `CollectionInfo` is a backend-agnostic metadata struct with `Name`, `PointCount`, `VectorSize`, and `Status` (mapped to `"green"`, `"yellow"`, `"red"`, or `"unknown"`).

## Data Types

### Point

Represents a vector point ready for storage:

```go
type Point struct {
    ID      string
    Vector  []float32
    Payload map[string]any
}
```

### SearchResult

Returned from vector similarity search:

```go
type SearchResult struct {
    ID      string
    Score   float32
    Payload map[string]any
}
```

### QueryResult

Higher-level result with typed metadata fields, returned from the query pipeline:

```go
type QueryResult struct {
    Text       string
    Source     string
    Section    string
    Category   string
    ChunkIndex int
    Score      float32
}
```

## Markdown Chunking

`ChunkMarkdown(text string, cfg ChunkConfig) []Chunk` is the primary chunking function. It also has an iterator variant, `ChunkMarkdownSeq`, for lazy evaluation.

### Configuration

```go
type ChunkConfig struct {
    Size    int // Target characters per chunk (default 500)
    Overlap int // Overlap in runes between adjacent chunks (default 50)
}
```

### Three-Level Splitting Strategy

1. **Section split** -- Text is first divided at `## ` header boundaries. Each header line is preserved with its section content. Top-level content before the first `## ` header forms its own section.

2. **Paragraph split** -- Sections larger than `Size` are split at double-newline boundaries. Multiple consecutive newlines are normalised to double-newlines before splitting.

3. **Sentence split** -- Paragraphs that individually exceed `Size` are split at sentence boundaries (`. `, `? `, `! `). Each sentence (or group of sentences that fits within `Size`) is treated as a separate sub-paragraph for accumulation. When no sentence boundaries exist, the oversized paragraph is added as-is.

Paragraphs are accumulated into chunks until adding the next paragraph would exceed `Size`. When the boundary is crossed, the current chunk is yielded and a new one starts.

### Overlap

When a chunk boundary is crossed, the new chunk begins with the trailing `Overlap` runes of the previous chunk. The overlap start point is aligned to the nearest word boundary (first space within the overlap slice) to avoid splitting mid-word. Overlap slicing is rune-safe, handling UTF-8 multi-byte characters correctly.

### Chunk Identity

Each `Chunk` carries:

- `Text` -- the chunk content
- `Section` -- the `## ` header title (empty if none)
- `Index` -- zero-based global counter across all sections in the document

`ChunkID(path string, index int, text string) string` produces a deterministic MD5 hash of `"path:index:first_100_runes"`. The resulting 32-character hex string is accepted by Qdrant's UUID parser as a point ID.

### Category Detection

`Category(path string) string` classifies files by keyword matching on the file path:

| Category | Path keywords |
|----------|---------------|
| `ui-component` | flux, ui/component |
| `brand` | brand, mascot |
| `product-brief` | brief |
| `help-doc` | help, draft |
| `task` | task, plan |
| `architecture` | architecture, migration |
| `documentation` | default fallback |

### Accepted File Types

`ShouldProcess(path string) bool` accepts `.md`, `.markdown`, `.pdf`, and `.txt` extensions.

## Ingestion Pipeline

### Directory Ingestion

`Ingest(ctx, store, embedder, cfg IngestConfig, progress IngestProgress) (*IngestStats, error)`:

1. Resolve and validate the source directory (must exist and be a directory).
2. Check whether the target collection exists. If `Recreate` is set and it exists, delete it first.
3. Create the collection if it does not exist, using `embedder.EmbedDimension()` to set the vector size.
4. Walk the directory recursively, collecting files that pass `ShouldProcess`.
5. For each file: read content, extracting text from PDFs when needed, call `ChunkMarkdown`, embed each chunk individually, and build `Point` structs.
6. Batch-upsert all accumulated points in slices of `BatchSize` (default 100).

The optional `IngestProgress` callback is invoked after each file is processed.

### Single-File Ingestion

`IngestFile(ctx, store, embedder, collection, filePath string, chunkCfg ChunkConfig) (int, error)` processes one file and returns the number of chunks stored.

### Point Payload Schema

Every stored point carries this payload:

| Field | Type | Description |
|-------|------|-------------|
| `text` | string | Raw chunk text |
| `source` | string | Relative file path from the ingestion directory root |
| `section` | string | Markdown section header (may be empty) |
| `category` | string | Category from `Category()` path detection |
| `chunk_index` | int | Chunk position within the document |

### Ingestion Configuration

```go
type IngestConfig struct {
    Directory  string
    Collection string      // Default: "hostuk-docs"
    Recreate   bool        // Delete and recreate the collection
    Verbose    bool        // Print per-file progress
    BatchSize  int         // Points per upsert batch (default 100)
    Chunk      ChunkConfig // Chunking parameters
}
```

## Query Pipeline

`Query(ctx, store, embedder, query string, cfg QueryConfig) ([]QueryResult, error)`:

1. Generate an embedding for the query text using `embedder.Embed`.
2. Construct a payload filter from `cfg.Category` if set (exact match on the `category` field).
3. Call `store.Search` with the query vector, limit, and filter.
4. Discard results with a score below `cfg.Threshold` (default 0.5).
5. Deserialise payload fields into typed `QueryResult` structs. The `chunk_index` field handles `int64`, `float64`, and `int` types to accommodate differences between Qdrant's protobuf response and JSON deserialisation.
6. If `cfg.Keywords` is true, apply keyword boosting (see below).

An iterator variant, `QuerySeq`, is also available.

### Query Configuration

```go
type QueryConfig struct {
    Collection string   // Default: "hostuk-docs"
    Limit      uint64   // Maximum results (default 5)
    Threshold  float32  // Minimum similarity score, 0-1 (default 0.5)
    Category   string   // Filter by category; empty means no filter
    Keywords   bool     // Enable keyword boosting post-filter
}
```

## Keyword Boosting

`KeywordFilter(results []QueryResult, keywords []string) []QueryResult` re-ranks results after vector search.

**Algorithm**: For each result, count how many keywords appear (case-insensitive substring match via `strings.Contains`) in the chunk text. Apply a 10% score boost per matching keyword:

```
score *= 1.0 + 0.1 * matchCount
```

Re-sort all results by boosted score in descending order.

**Keyword extraction**: `extractKeywords(query string)` splits the query on whitespace and discards words shorter than 3 characters. This removes common articles and prepositions that would produce false-positive matches.

When `cfg.Keywords` is true, the `Query` function automatically extracts keywords from the query string and applies the filter after the threshold filter.

## Result Formatting

Three output formatters are provided:

| Function | Format | Use Case |
|----------|--------|----------|
| `FormatResultsText` | Plain text with score/source/section headers | Human-readable terminal output |
| `FormatResultsContext` | XML `<retrieved_context>` with `<document>` elements | LLM prompt injection |
| `FormatResultsJSON` | JSON array with source, section, category, score, text | Structured consumption by other tools |

`FormatResultsContext` applies `html.EscapeString` to all attribute values and text content to produce well-formed XML safe for embedding in prompts:

```xml
<retrieved_context>
<document source="api/rate-limiting.md" section="Configuration" category="architecture">
Rate limiting is configured per-route using the throttle middleware...
</document>
</retrieved_context>
```

## Ollama Embedding Client

`OllamaClient` wraps the `github.com/ollama/ollama/api` HTTP client.

- **Connection**: HTTP on port 11434, 30-second timeout.
- **Embedding**: Calls `/api/embed`. The Ollama API returns `float64` values; these are converted to `float32` for Qdrant compatibility.
- **Batch embedding**: `EmbedBatch` calls `Embed` for each input in order and preserves input order in the response.
- **Model verification**: `VerifyModel` sends a test embedding request to confirm the model is loaded.

Supported models and their dimensions:

| Model | Dimensions |
|-------|-----------|
| `nomic-embed-text` (default) | 768 |
| `mxbai-embed-large` | 1024 |
| `all-minilm` | 384 |
| Unknown models | 768 (fallback) |

## Qdrant Vector Store Client

`QdrantClient` wraps the `github.com/qdrant/go-client` gRPC library.

- **Connection**: gRPC on port 6334. Supports TLS and API key authentication via `QdrantConfig`.
- **Collection creation**: Uses cosine distance metric. Vector dimensionality is derived from `Embedder.EmbedDimension()`.
- **Point IDs**: Must be valid UUIDs. `ChunkID()` produces MD5 hex strings that Qdrant accepts.
- **Search**: Uses Qdrant's `QueryPoints` API with optional `Must` filter conditions (logical AND). Results include similarity score and full payload.
- **Payload conversion**: Qdrant protobuf `Value` types are converted to native Go types (`string`, `int64`, `float64`, `bool`, `[]any`, `map[string]any`) by the internal `valueToGo` function.

## Convenience Helpers

Two tiers of helpers are provided in `helpers.go`:

### Interface-accepting (`*With` variants)

Accept pre-constructed `VectorStore` and `Embedder`. Suitable for long-lived processes, server contexts, and high-throughput use where you want to reuse connections:

```go
QueryWith(ctx, store, embedder, question, collectionName string, topK int) ([]QueryResult, error)
QueryContextWith(ctx, store, embedder, question, collectionName string, topK int) (string, error)
IngestDirWith(ctx, store, embedder, directory, collectionName string, recreate bool) error
IngestFileWith(ctx, store, embedder, filePath, collectionName string) (int, error)
```

### Default-client wrappers

Construct new `QdrantClient` and `OllamaClient` on each call using `DefaultQdrantConfig()` and `DefaultOllamaConfig()`. Each call opens and closes a gRPC connection. Suitable for CLI commands and infrequent operations:

```go
QueryDocs(ctx, question, collectionName string, topK int) ([]QueryResult, error)
QueryDocsContext(ctx, question, collectionName string, topK int) (string, error)
IngestDirectory(ctx, directory, collectionName string, recreate bool) error
IngestSingleFile(ctx, filePath, collectionName string) (int, error)
```

`IngestDirectory` and `IngestSingleFile` additionally run `HealthCheck` on Qdrant and `VerifyModel` on Ollama before proceeding.

## Collection Management

Package-level helpers in `collections.go` delegate to `VectorStore`:

```go
ListCollections(ctx, store VectorStore) ([]string, error)
ListCollectionsSeq(ctx, store VectorStore) (iter.Seq[string], error)
DeleteCollection(ctx, store VectorStore, name string) error
CollectionStats(ctx, store VectorStore, name string) (*CollectionInfo, error)
```

## Data Flow

```
                    Ingestion Pipeline
                    ==================

  Directory of .md/.txt files
         |
         v
  filepath.WalkDir (filter by ShouldProcess)
         |
         v
  ChunkMarkdown (sections -> paragraphs -> sentences)
         |
         v
  Embedder.Embed (per chunk)
         |
         v
  Point{ID: ChunkID(), Vector, Payload}
         |
         v
  VectorStore.UpsertPoints (batched)


                    Query Pipeline
                    ==============

  User question (string)
         |
         v
  Embedder.Embed (generate query vector)
         |
         v
  VectorStore.Search (cosine similarity, optional category filter)
         |
         v
  Threshold filter (discard score < 0.5)
         |
         v
  KeywordFilter (optional, +10% per keyword match)
         |
         v
  FormatResults{Text,Context,JSON}
```

## Performance Characteristics

Measured on AMD Ryzen 9 9950X with RX 7800 XT (ROCm), `nomic-embed-text` (F16), Qdrant v1.16.3 (Docker):

| Operation | Latency | Throughput |
|-----------|---------|------------|
| Single embed | 10.3ms | 97/sec |
| Batch embed (10 texts) | 102ms | 98/sec effective |
| Qdrant search (100 points) | 111us | 9,042 QPS |
| Qdrant search (200 points) | 152us | 6,580 QPS |
| Chunk 50 sections | 11.2us | 89K/sec |
| Chunk 1000 paragraphs | 107us | 9.4K/sec |

The embedding step dominates pipeline latency. In a full ingest-then-query cycle for 5 documents, approximately 95% of elapsed time is spent in embedding calls. Text length (50 to 2000 characters) has negligible effect on embedding latency because tokenisation and HTTP overhead dominate the GPU compute time (~2ms per embed).
