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
