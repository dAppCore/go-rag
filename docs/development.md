# go-rag Development Guide

## Prerequisites

### Required Services

**Qdrant** — vector database, gRPC on port 6334:

```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  qdrant/qdrant:v1.16.3
```

REST is on 6334, gRPC on 6334. The library connects via gRPC only.

**Ollama** — embedding model server, HTTP on port 11434:

```bash
# Install Ollama (see https://ollama.com for platform-specific instructions)
ollama pull nomic-embed-text
ollama serve
```

For AMD GPU (ROCm) on Linux, install the ROCm-enabled Ollama binary from the Ollama releases page. The `nomic-embed-text` model (274MB F16) is the default and recommended model.

### Go Version

Go 1.25 or later. The module uses a Go workspace (`go.work`) at the repository root that includes the `forge.lthn.ai/core/go` dependency via a local `replace` directive:

```
replace forge.lthn.ai/core/go => ../go
```

Ensure `../go` (the `go-rag` sibling repository) is present and its `go.mod` is consistent.

## Build and Test

### Unit Tests (no external services required)

```bash
go test ./...
```

Runs 135 tests covering all pure functions and mock-based integration. No Qdrant or Ollama instance needed.

### Integration Tests (require live Qdrant and Ollama)

```bash
go test -tags rag ./...
```

Runs the full test suite of 204 tests, including:

- `qdrant_integration_test.go` — 11 subtests (collection lifecycle, upsert, search, filter)
- `ollama_integration_test.go` — 9 subtests (model verification, single/batch embed, determinism)
- `integration_test.go` — 12 subtests (full pipeline, all helpers, semantic similarity)

### Running a Single Test

```bash
go test -v -run TestChunkMarkdown ./...
go test -v -tags rag -run TestIntegration_FullPipeline ./...
```

### Benchmarks

```bash
# Mock-only benchmarks (no services):
go test -bench=. -benchmem ./...

# GPU/service benchmarks (requires Qdrant + Ollama):
go test -tags rag -bench=. -benchmem ./...
```

Key benchmarks: `BenchmarkChunk`, `BenchmarkChunkWithOverlap`, `BenchmarkQuery_Mock`, `BenchmarkIngest_Mock`, `BenchmarkFormatResults`, `BenchmarkKeywordFilter`, `BenchmarkEmbedSingle`, `BenchmarkEmbedBatch`, `BenchmarkQdrantSearch`, `BenchmarkFullPipeline`.

### Coverage

```bash
# Mock-only coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Full coverage with services:
go test -tags rag -coverprofile=coverage.out ./...
```

Current coverage targets: 69.0% without services, 89.2% with live services.

## Test Patterns

### Build Tag Strategy

Tests requiring external services carry the `//go:build rag` build tag. This isolates them from CI environments that lack live services. All pure-function and mock-based tests have no build tag and run unconditionally.

```go
//go:build rag

package rag
```

### Mock Implementations

`mock_test.go` provides two in-package test doubles:

**`mockEmbedder`** — deterministic, all-0.1 vectors of configurable dimension. Supports error injection via `embedErr`/`batchErr` and custom behaviour via `embedFunc`. Tracks all `Embed` and `EmbedBatch` calls with a mutex for concurrent-safe counting.

**`mockVectorStore`** — in-memory map-backed store. Returns stored points on search with fake descending scores (`1.0, 0.9, 0.8, ...`). Supports per-method error injection and a custom `searchFunc` override. Tracks all method calls.

Constructors:

```go
embedder := newMockEmbedder(768)
store := newMockVectorStore()
```

Error injection:

```go
embedder.embedErr = errors.New("embed failed")
store.upsertErr = errors.New("store unavailable")
```

### Test Naming Convention

Tests use `_Good`, `_Bad`, `_Ugly` suffix semantics:

- `_Good` — happy path
- `_Bad` — expected error conditions (invalid input, service errors)
- `_Ugly` — panic or edge cases

Table-driven subtests are used for pure functions with many input variants (e.g., `valueToGo`, `EmbedDimension`, `FormatResults*`).

### Integration Test Patterns

Integration tests skip gracefully when services are unavailable. Qdrant integration tests call `HealthCheck` and skip if it fails:

```go
if err := client.HealthCheck(ctx); err != nil {
    t.Skipf("Qdrant unavailable: %v", err)
}
```

**Indexing latency**: After upserting points to Qdrant, tests include a 500ms sleep before searching to avoid flaky results on slower machines.

**Point ID format**: Qdrant requires UUID-format point IDs. Use `ChunkID()` to generate IDs, or any 32-character lowercase hex string. Arbitrary strings (e.g., `"point-alpha"`) are rejected by Qdrant's UUID parser.

**Unique collection names**: Integration tests create collections with timestamped or randomised names and delete them in `t.Cleanup` to avoid cross-test interference.

## Coding Standards

### Language

UK English throughout — in comments, documentation, variable names, and error messages. Use: `colour`, `organisation`, `initialise`, `serialise`, `behaviour`, `recognised`. Do not use American spellings.

### Go Style

- `declare(strict_types=1)` equivalent: all functions have explicit parameter and return types.
- Error messages use the `log.E("component.Method", "what failed", err)` pattern from `forge.lthn.ai/core/go/pkg/log`. This wraps errors with component context for structured logging.
- No naked returns.
- Exported types and functions have doc comments.
- Internal helpers are unexported and documented with concise inline comments.

### Formatting

Standard `gofmt` / `goimports`. No additional linter configuration is defined; the project follows standard Go conventions.

### Licence

Every new Go source file must include the EUPL-1.2 licence header:

```go
// Copyright (C) 2026 Host UK Ltd.
// SPDX-License-Identifier: EUPL-1.2
```

### Commits

Conventional commits format: `type(scope): description`

Common types: `feat`, `fix`, `test`, `refactor`, `docs`, `chore`.

Every commit must include the co-author trailer:

```
Co-Authored-By: Virgil <virgil@lethean.io>
```

Example:

```
feat(chunk): add sentence-aware splitting for oversized paragraphs

When a paragraph exceeds ChunkConfig.Size, split at sentence boundaries
(". ", "? ", "! ") rather than adding the whole paragraph as an
oversized chunk. Falls back to the full paragraph when no sentence
boundaries exist.

Co-Authored-By: Virgil <virgil@lethean.io>
```

## Adding a New Embedding Provider

1. Create a new file, e.g., `openai.go`.
2. Define a config struct and constructor.
3. Implement the `Embedder` interface: `Embed`, `EmbedBatch`, `EmbedDimension`.
4. Add a corresponding `openai_test.go` with pure-function tests (config defaults, dimension lookup).
5. Add an `openai_integration_test.go` with the `//go:build rag` tag for live API tests.

## Adding a New Vector Backend

1. Create a new file, e.g., `weaviate.go`.
2. Define a config struct and constructor.
3. Implement all methods of the `VectorStore` interface.
4. Ensure `CollectionInfo` maps backend status codes to the `green`/`yellow`/`red`/`unknown` strings.
5. Add integration tests under the `rag` build tag.

## Common Pitfalls

**Wrong replace path in go.mod**: The replace directive must point to `../go`, not `../core`. If `go test` reports module not found for `forge.lthn.ai/core/go`, verify the `replace` directive and that the sibling directory exists.

**Qdrant UUID requirement**: Do not pass arbitrary strings as point IDs. Always use `ChunkID()` or another MD5/UUID generator. Qdrant rejects non-UUID strings with `Unable to parse UUID: <value>`.

**EmbedBatch is sequential**: There is no batch endpoint in the Ollama API. `EmbedBatch` calls `Embed` in a loop. If throughput is critical, parallelise calls yourself with goroutines and limit concurrency to avoid overwhelming the Ollama process.

**Collection not created before upsert**: `Ingest` handles collection creation automatically. If calling `UpsertPoints` directly, call `CreateCollection` (or `CollectionExists` + conditional create) first.

**Score threshold**: The default threshold is 0.5. For short or ambiguous queries this may return zero results. Lower the threshold in `QueryConfig.Threshold` or set it to 0.0 to return all results above the `Limit`.

**Convenience wrappers open a new connection per call**: `QueryDocs`, `IngestDirectory`, and `IngestSingleFile` construct new `QdrantClient` instances (and new gRPC connections) on every invocation. Use the `*With` variants with pre-created clients for server processes or loops.
