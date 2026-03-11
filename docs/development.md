---
title: Development Guide
description: How to set up, build, test, and contribute to go-rag — prerequisites, test patterns, coding standards, and extension points.
---

# Development Guide

This document covers everything needed to work on `forge.lthn.ai/core/go-rag`: setting up the required services, running tests, understanding the test architecture, and following the project's coding standards.

## Prerequisites

### Go Version

Go 1.26 or later. The module is part of a Go workspace (`go.work`) that resolves `forge.lthn.ai/core/*` dependencies via local paths. Ensure the sibling modules referenced in your workspace file are present and their `go.mod` files are consistent.

### External Services

Two services are required for integration tests. Unit tests and mock-based tests run without either.

**Qdrant** -- vector database, gRPC on port 6334:

```bash
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  qdrant/qdrant:v1.16.3
```

Port 6333 is the REST API (not used by the library). Port 6334 is gRPC (used by the library).

**Ollama** -- embedding model server, HTTP on port 11434:

```bash
# Install Ollama from https://ollama.com
ollama pull nomic-embed-text
ollama serve
```

The `nomic-embed-text` model (274MB, F16) is the default. For AMD GPUs with ROCm, install the ROCm-enabled Ollama binary from the Ollama releases page.

## Build and Test

### Unit Tests (no external services)

```bash
go test ./...
```

Runs all pure-function and mock-based tests. No Qdrant or Ollama instance is needed.

### Integration Tests (require live Qdrant and Ollama)

```bash
go test -tags rag ./...
```

Runs the full suite including:

- `qdrant_integration_test.go` -- collection lifecycle, upsert, search, payload filtering
- `ollama_integration_test.go` -- model verification, single and batch embedding, determinism
- `integration_test.go` -- full pipeline, all helper variants, semantic similarity verification

Integration tests skip gracefully when services are unavailable (they call `HealthCheck` and `t.Skipf` on failure).

### Running a Single Test

```bash
go test -v -run TestChunkMarkdown ./...
go test -v -tags rag -run TestIntegration_FullPipeline ./...
```

### Benchmarks

```bash
# Mock-only benchmarks (no services needed):
go test -bench=. -benchmem ./...

# GPU/service benchmarks (require Qdrant + Ollama):
go test -tags rag -bench=. -benchmem ./...
```

Key benchmarks include `BenchmarkChunk`, `BenchmarkChunkWithOverlap`, `BenchmarkQuery_Mock`, `BenchmarkIngest_Mock`, `BenchmarkFormatResults`, `BenchmarkKeywordFilter`, `BenchmarkEmbedSingle`, `BenchmarkEmbedBatch`, `BenchmarkQdrantSearch`, and `BenchmarkFullPipeline`.

### Test Coverage

```bash
# Mock-only coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Full coverage with live services:
go test -tags rag -coverprofile=coverage.out ./...
```

Coverage targets: ~69% without services, ~89% with live services.

### Linting

```bash
golangci-lint run ./...
go vet ./...
gofmt -w .
```

The `.golangci.yml` configuration enables `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`, `typecheck`, `gocritic`, and `gofmt`.

## Test Architecture

### Build Tag Strategy

Tests requiring external services carry the `//go:build rag` build tag:

```go
//go:build rag

package rag
```

This isolates them from CI environments that lack live services. All pure-function and mock-based tests have no build tag and run unconditionally with `go test ./...`.

### Mock Implementations

`mock_test.go` provides two in-package test doubles:

**`mockEmbedder`** -- returns deterministic all-0.1 vectors of configurable dimension. Features:

- Call tracking: `embedCalls` records every text passed to `Embed`; `batchCalls` records `EmbedBatch` inputs
- Error injection: set `embedErr` or `batchErr` to force failures
- Custom behaviour: set `embedFunc` for per-test logic
- Thread-safe: all state is guarded by a mutex

**`mockVectorStore`** -- in-memory map-backed store. Features:

- Stores points per collection in `map[string][]Point`
- Search returns stored points with fake descending scores (1.0, 0.9, 0.8, ...)
- Supports payload filter matching (exact string comparison)
- Per-method error injection: `createErr`, `existsErr`, `deleteErr`, `listErr`, `infoErr`, `upsertErr`, `searchErr`
- Custom search: set `searchFunc` to override default behaviour
- Call tracking for all methods

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

- `_Good` -- happy path
- `_Bad` -- expected error conditions (invalid input, service errors)
- `_Ugly` -- panic or edge cases

Table-driven subtests are used for pure functions with many input variants (e.g., `valueToGo`, `EmbedDimension`, `FormatResults*`).

### Integration Test Patterns

**Graceful skip**: Integration tests call `HealthCheck` and skip if the service is unavailable:

```go
if err := client.HealthCheck(ctx); err != nil {
    t.Skipf("Qdrant unavailable: %v", err)
}
```

**Indexing latency**: After upserting points to Qdrant, tests include a 500ms sleep before searching to account for Qdrant's indexing delay.

**Point ID format**: Qdrant requires UUID-format point IDs. Always use `ChunkID()` to generate IDs. Arbitrary strings like `"point-alpha"` are rejected by Qdrant's UUID parser.

**Collection isolation**: Integration tests create collections with timestamped or randomised names and delete them in `t.Cleanup` to avoid cross-test interference.

## Coding Standards

### Language

UK English throughout -- in comments, documentation, variable names, and error messages. Use `colour`, `organisation`, `initialise`, `serialise`, `behaviour`, `recognised`. Do not use American spellings.

### Error Handling

Error messages use the `log.E("component.Method", "what failed", err)` pattern from `forge.lthn.ai/core/go-log`. This wraps errors with component context for structured logging:

```go
return log.E("rag.Ingest", "error resolving directory", err)
```

### Go Style

- All functions have explicit parameter and return types
- No naked returns
- Exported types and functions have doc comments
- Internal helpers are unexported with concise inline comments
- Standard `gofmt` / `goimports` formatting

### Licence Header

Every new Go source file should include:

```go
// Copyright (C) 2026 Host UK Ltd.
// SPDX-License-Identifier: EUPL-1.2
```

### Commit Messages

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

1. Create a new file (e.g., `openai.go`) with a config struct and constructor.
2. Implement the `Embedder` interface: `Embed`, `EmbedBatch`, `EmbedDimension`.
3. Add a unit test file (`openai_test.go`) covering config defaults and dimension lookup.
4. Add an integration test file (`openai_integration_test.go`) with the `//go:build rag` tag for live API tests.

## Adding a New Vector Backend

1. Create a new file (e.g., `weaviate.go`) with a config struct and constructor.
2. Implement all methods of the `VectorStore` interface.
3. Ensure `CollectionInfo` maps backend-specific status codes to the `"green"` / `"yellow"` / `"red"` / `"unknown"` convention.
4. Add integration tests under the `//go:build rag` tag.

## Common Pitfalls

**Qdrant UUID requirement**: Do not pass arbitrary strings as point IDs. Always use `ChunkID()` or another MD5/UUID generator. Qdrant rejects non-UUID strings with `Unable to parse UUID: <value>`.

**EmbedBatch is sequential**: There is no batch endpoint in the Ollama API. `EmbedBatch` calls `Embed` in a loop. For higher throughput, parallelise calls with goroutines and limit concurrency to avoid overwhelming the Ollama process.

**Collection must exist before upsert**: `Ingest` handles collection creation automatically. If calling `UpsertPoints` directly, create the collection first (or use `CollectionExists` to check).

**Score threshold filtering**: The default threshold is 0.5. Short or ambiguous queries may return zero results. Lower `QueryConfig.Threshold` or set it to 0.0 to return all results up to the limit.

**Convenience wrappers open connections per call**: `QueryDocs`, `IngestDirectory`, and `IngestSingleFile` construct a new `QdrantClient` (and gRPC connection) on every invocation. Use the `*With` variants with pre-created clients for server processes or loops.

**EmbedDimension fallback**: Unknown model names return 768 (the `nomic-embed-text` dimension). If a model with a different dimension is configured and its dimension is not known to the library, the collection will be created with an incorrect vector size, causing upsert failures at the Qdrant level.

**Workspace module resolution**: The `go.mod` may contain `replace` directives for local development. Ensure the referenced sibling directories exist and their `go.mod` files are consistent. If `go test` reports module-not-found errors for `forge.lthn.ai/core/*`, verify the workspace configuration.
