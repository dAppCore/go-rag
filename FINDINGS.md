# FINDINGS.md — go-rag Research & Discovery

## 2026-02-19: Split from go-ai (Virgil)

### Origin

Extracted from `forge.lthn.ai/core/go-ai/rag/`. Zero internal go-ai dependencies.

### What Was Extracted

- 7 Go files (~1,017 LOC excluding tests)
- 1 test file (chunk_test.go)

### Key Finding: Minimal Test Coverage

Only chunk.go has tests. The Qdrant and Ollama clients are untested — they depend on external services (Qdrant server, Ollama API) which makes unit testing harder. Consider mock interfaces.

### Consumers

- `go-ai/ai/rag.go` wraps this as `QueryRAGForTask()` facade
- `go-ai/mcp/tools_rag.go` exposes RAG as MCP tools

---

## 2026-02-19: Environment Review (Charon)

### go.mod Fix

Replace directive was `../core` — should be `../go`. Fixed. Tests now pass.

### Coverage

```
go-rag: 18.4% coverage (only chunk.go tested)
```

### Infrastructure Status

| Service | Status | Notes |
|---------|--------|-------|
| Qdrant | **Not running** | Need `docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant` |
| Ollama | **Not running locally** | M3 has Ollama at 10.69.69.108:11434, but local install preferred for tests |

### Testability Analysis

| File | Lines | Testable Without Services | Notes |
|------|-------|---------------------------|-------|
| chunk.go | 205 | Yes — pure functions | 8 tests exist, good coverage |
| query.go | 163 | **Partially** — FormatResults* are pure | Query() needs Qdrant + Ollama |
| qdrant.go | 226 | No — all gRPC calls | Need live Qdrant or mock interface |
| ollama.go | 120 | **Partially** — EmbedDimension is pure | Embed() needs live Ollama |
| ingest.go | 217 | No — orchestrates Qdrant + Ollama | Need mocks or live services |
| helpers.go | 89 | **Partially** — QueryDocs/IngestDirectory are convenience wrappers | Same deps as query/ingest |

### Recommendation

Phase 1 should focus on pure-function tests (FormatResults*, EmbedDimension, defaults, valueToGo). Phase 2 extracts `Embedder` and `VectorStore` interfaces to enable mocked testing for ingest/query. Phase 3+ needs live services.

---

## 2026-02-20: Phase 1 Pure-Function Tests Complete (go-rag agent)

### Coverage Improvement

```
Before: 18.4% (8 tests in chunk_test.go only)
After:  38.8% (66 tests across 4 test files)
```

### Per-Function Coverage

All targeted pure functions now at 100% coverage:

| Function | File | Coverage |
|----------|------|----------|
| FormatResultsText | query.go | 100% |
| FormatResultsContext | query.go | 100% |
| FormatResultsJSON | query.go | 100% |
| DefaultQueryConfig | query.go | 100% |
| DefaultOllamaConfig | ollama.go | 100% |
| DefaultQdrantConfig | qdrant.go | 100% |
| DefaultChunkConfig | chunk.go | 100% |
| DefaultIngestConfig | ingest.go | 100% |
| EmbedDimension | ollama.go | 100% |
| Model | ollama.go | 100% |
| valueToGo | qdrant.go | 100% |
| ChunkID | chunk.go | 100% |
| ChunkMarkdown | chunk.go | 97.6% |
| pointIDToString | qdrant.go | 83.3% |

### Discoveries

1. **OllamaClient can be constructed with nil `client` field** for testing pure methods (EmbedDimension, Model). The struct fields are unexported but accessible within the same package.

2. **Qdrant protobuf constructors** (`NewValueString`, `NewValueInt`, etc.) make it straightforward to build test values for `valueToGo` without needing a live Qdrant connection.

3. **pointIDToString default branch** (83.3%) — the uncovered path is a `PointId` with `PointIdOptions` set to an unknown type. This cannot be constructed via the public API (`NewIDNum` and `NewIDUUID` are the only constructors), so the 83.3% is the practical maximum without reflection hacks.

4. **FormatResultsJSON output is valid JSON** — confirmed by round-tripping through `json.Unmarshal` in tests. The hand-crafted JSON builder in `query.go` correctly handles escaping of special characters.

5. **ChunkMarkdown rune safety** — the overlap logic in `chunk.go` correctly uses `[]rune` slicing, confirmed by CJK text tests that would corrupt if byte-level slicing were used.

6. **Remaining 61.2% untested** is entirely in functions that require live Qdrant or Ollama: `NewQdrantClient`, `Search`, `UpsertPoints`, `Embed`, `EmbedBatch`, `Ingest`, `IngestFile`, and the helper wrappers. These are Phase 2 (mock interfaces) and Phase 3 (integration) targets.

### Test Files Created

| File | Tests | What It Covers |
|------|-------|----------------|
| query_test.go | 18 | FormatResultsText, FormatResultsContext, FormatResultsJSON, DefaultQueryConfig |
| ollama_test.go | 8 | DefaultOllamaConfig, EmbedDimension (5 models), Model |
| qdrant_test.go | 24 | DefaultQdrantConfig, pointIDToString, valueToGo (all types + nesting), Point, SearchResult |
| chunk_test.go (extended) | 16 new | Empty input, headers-only, unicode/emoji, long paragraphs, config boundaries, ChunkID edge cases, DefaultChunkConfig, DefaultIngestConfig |

---

## 2026-02-20: Phase 2 Test Infrastructure Complete (go-rag agent)

### Coverage Improvement

```
Before: 38.8% (66 tests across 4 test files)
After:  69.0% (135 leaf-level tests across 7 test files)
```

### Interface Extraction

Two interfaces extracted to decouple business logic from external services:

| Interface | File | Methods | Satisfied By |
|-----------|------|---------|--------------|
| `Embedder` | embedder.go | Embed, EmbedBatch, EmbedDimension | `*OllamaClient` |
| `VectorStore` | vectorstore.go | CreateCollection, CollectionExists, DeleteCollection, UpsertPoints, Search | `*QdrantClient` |

### Signature Changes

The following functions now accept interfaces instead of concrete types:

| Function | Old Signature | New Signature |
|----------|--------------|---------------|
| `Ingest` | `*QdrantClient, *OllamaClient` | `VectorStore, Embedder` |
| `IngestFile` | `*QdrantClient, *OllamaClient` | `VectorStore, Embedder` |
| `Query` | `*QdrantClient, *OllamaClient` | `VectorStore, Embedder` |

These changes are backwards-compatible because `*QdrantClient` satisfies `VectorStore` and `*OllamaClient` satisfies `Embedder`.

### New Helper Functions

Added interface-accepting helpers that the convenience wrappers now delegate to:

| Function | Purpose |
|----------|---------|
| `QueryWith` | Query with provided store + embedder |
| `QueryContextWith` | Query + format as context XML |
| `IngestDirWith` | Ingest directory with provided store + embedder |
| `IngestFileWith` | Ingest single file with provided store + embedder |

### Per-Function Coverage (Phase 2 targets)

| Function | File | Coverage | Notes |
|----------|------|----------|-------|
| Ingest | ingest.go | 86.8% | Uncovered: filepath.Rel error branch |
| IngestFile | ingest.go | 100% | |
| Query | query.go | 100% | |
| QueryWith | helpers.go | 100% | |
| QueryContextWith | helpers.go | 100% | |
| IngestDirWith | helpers.go | 100% | |
| IngestFileWith | helpers.go | 100% | |

### Discoveries

1. **Interface method signatures must match exactly** -- `EmbedDimension()` returns `uint64` (not `int`), and `Search` takes `limit uint64` and `filter map[string]string` (not `limit int, threshold float32`). The task description suggested approximate signatures; the actual code was the source of truth.

2. **Convenience wrappers cannot be mocked** -- `QueryDocs`, `IngestDirectory`, `IngestSingleFile` construct their own concrete clients internally. Added `*With` variants that accept interfaces for testability. The convenience wrappers now delegate to these.

3. **ChunkMarkdown preserves section headers in chunk text** -- Small sections that fit within the chunk size include the `## Header` line in the chunk text. Tests must use `Contains` rather than `Equal` when checking chunk text.

4. **Mock vector store score calculation** -- The mock assigns scores as `1.0 - index*0.1`, so the second stored point gets 0.9. Tests using threshold must account for this.

5. **Remaining 31% untested** is entirely in concrete client implementations (QdrantClient methods, OllamaClient.Embed/EmbedBatch, NewOllamaClient, NewQdrantClient) and the convenience wrapper functions that construct live clients. These are Phase 3 (integration test with live services) targets.

### Test Files Created/Modified

| File | New Tests | What It Covers |
|------|-----------|----------------|
| mock_test.go | -- | mockEmbedder + mockVectorStore implementations |
| ingest_test.go (new) | 23 | Ingest (17 subtests) + IngestFile (6 subtests) with mocks |
| query_test.go (extended) | 12 | Query function with mocks: embedding, search, threshold, errors, payload extraction |
| helpers_test.go (new) | 16 | QueryWith (4), QueryContextWith (3), IngestDirWith (4), IngestFileWith (5) |

### New Source Files

| File | Purpose |
|------|---------|
| embedder.go | `Embedder` interface definition |
| vectorstore.go | `VectorStore` interface definition |

---

## 2026-02-20: Phase 3 Integration Tests with Live Services (go-rag agent)

### Coverage Improvement

```
Before: 69.0% (135 tests across 7 test files, mock-based only)
After:  89.2% (204 tests across 10 test files, includes live Qdrant + Ollama)
```

### Infrastructure Verified

| Service | Version | Status | Connection |
|---------|---------|--------|------------|
| Qdrant | 1.16.3 | Running (Docker) | gRPC localhost:6334, REST localhost:6333 |
| Ollama | native + ROCm | Running | HTTP localhost:11434, model: nomic-embed-text (F16, 274MB) |

### Discoveries

1. **Qdrant point IDs must be valid UUIDs** -- `qdrant.NewID()` wraps the string as a UUID field. Qdrant's server-side UUID parser accepts 32-character hex strings (as produced by `ChunkID` via MD5) but rejects arbitrary strings like `point-alpha`. Error: `Unable to parse UUID: point-alpha`. Integration tests must use `ChunkID()` or MD5 hex format for point IDs.

2. **Qdrant Go client version warning is benign** -- The client library (v1.16.2) logs `WARN Unable to compare versions` and `Client version is not compatible with server version` when connecting to Qdrant v1.16.3. This is a cosmetic mismatch in version parsing — all operations function correctly despite the warning.

3. **Qdrant indexing latency** -- After upserting points, a 500ms sleep is needed before searching to avoid flaky results. For small datasets the indexing is nearly instant, but the sleep provides a safety margin on slower machines.

4. **Ollama embedding determinism** -- Embedding the same text twice with `nomic-embed-text` produces bit-identical vectors (`float32` level). This is important for idempotent ingest operations.

5. **Ollama accepts empty strings** -- `Embed(ctx, "")` returns a valid 768-dimension vector without error. This is Ollama-specific behaviour and may differ with other embedding providers.

6. **Semantic similarity works as expected** -- When ingesting both programming and cooking documents, a query about "Go functions and closures" correctly ranks the programming document highest. The cosine distance metric in Qdrant combined with nomic-embed-text embeddings provides meaningful semantic differentiation.

7. **Convenience wrappers (QueryDocs, IngestDirectory) create their own gRPC connections** -- Each call to `QueryDocs` or `IngestDirectory` establishes a new Qdrant gRPC connection. In production this is fine for CLI commands, but for high-throughput scenarios the `*With` variants that accept pre-created clients should be preferred.

8. **Remaining ~11% untested** -- The uncovered code is primarily error-handling branches in `NewQdrantClient` (connection failure), `Close()`, and the `filepath.Rel` error branch in `Ingest`. These represent defensive code paths that are difficult to trigger in normal operation.

### Test Files Created

| File | Tests | What It Covers |
|------|-------|----------------|
| qdrant_integration_test.go | 11 | Health check, create/delete/list/info collection, exists check, upsert+search, filter, empty upsert, ID validation, overwrite |
| ollama_integration_test.go | 9 | Verify model, embed single, embed batch, consistency, dimension match, model name, different texts, non-zero values, empty string |
| integration_test.go | 12 | End-to-end ingest+query, format results, IngestFile, QueryWith, QueryContextWith, IngestDirWith, IngestFileWith, QueryDocs, IngestDirectory, recreate flag, semantic similarity |

### Build Tag Strategy

All integration tests use `//go:build rag` to isolate them from CI runs that lack live services:

```bash
go test ./... -count=1               # 135 tests, 69.0% — mock-only, no services needed
go test -tags rag ./... -count=1     # 204 tests, 89.2% — requires Qdrant + Ollama
```
