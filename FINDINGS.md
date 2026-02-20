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
