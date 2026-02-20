# TODO.md — go-rag Task Queue

Dispatched from core/go orchestration. Pick up tasks in phase order.

---

## Phase 0: Environment Setup

- [x] **Fix go.mod replace directive** — Was `../core`, corrected to `../go`. Commit and push. (Charon, 19 Feb 2026)
- [x] **Run Qdrant locally** — Docker: `docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant`. Test with `curl http://localhost:6334/healthz`.
- [x] **Install Ollama** — `curl -fsSL https://ollama.com/install.sh | sh`. Pull embedding model: `ollama pull nomic-embed-text`.
- [x] **Verify both services** — Both running on snider-linux.

## Phase 1: Unit Tests (18.4% -> 38.8% coverage)

All pure-function tests complete. Remaining untested functions require live services (Phase 2/3).

### Testable Without External Services

- [x] **FormatResults tests** — FormatResultsText, FormatResultsContext, FormatResultsJSON with known QueryResult inputs. Pure string formatting, no deps. (acb987a)
- [x] **DefaultConfig tests** — Verify DefaultQdrantConfig, DefaultOllamaConfig, DefaultQueryConfig, DefaultChunkConfig, DefaultIngestConfig return expected values. (acb987a)
- [x] **EmbedDimension tests** — OllamaClient.EmbedDimension() for each model name (nomic-embed-text=768, mxbai-embed-large=1024, all-minilm=384, unknown=768). (acb987a)
- [x] **Point/SearchResult types** — Round-trip tests for Point struct and pointIDToString helper. (acb987a)
- [x] **valueToGo tests** — Qdrant value conversion for string, int, double, bool, list, struct, nil. (acb987a)
- [x] **Additional chunk tests** — Empty input, only headers no content, unicode/emoji, very long paragraph. (acb987a)

### Require External Services (use build tag `//go:build rag`)

- [x] **Qdrant client tests** — Create collection, upsert, search, delete, list, info, filter, overwrite. Skip if Qdrant unavailable. 11 subtests in `qdrant_integration_test.go`. (e90f281)
- [x] **Ollama client tests** — Embed single text, embed batch, verify model, consistency, dimension check, different texts, non-zero values, empty string. 9 subtests in `ollama_integration_test.go`. (e90f281)
- [x] **Full pipeline integration test** — Ingest directory, query, format results, all helpers (QueryWith, QueryContextWith, IngestDirWith, IngestFileWith, QueryDocs, IngestDirectory), recreate flag, semantic similarity. 12 subtests in `integration_test.go`. (e90f281)

## Phase 2: Test Infrastructure (38.8% -> 69.0% coverage)

- [x] **Interface extraction** — Extracted `Embedder` interface (embedder.go) and `VectorStore` interface (vectorstore.go). Updated `Ingest`, `IngestFile`, `Query` to accept interfaces. Added `QueryWith`, `QueryContextWith`, `IngestDirWith`, `IngestFileWith` helpers. (a49761b)
- [x] **Mock embedder** — Returns deterministic 0.1 vectors, tracks all calls, supports error injection and custom embed functions. (a49761b)
- [x] **Mock vector store** — In-memory map, stores points, returns them on search with fake descending scores, supports filtering, tracks all calls. (a49761b)
- [x] **Re-test with mocks** — 69 new mock-based tests across ingest (23), query (12), and helpers (16). Coverage from 38.8% to 69.0%. (a49761b)

## Phase 3: Enhancements

All tasks are pure Go, testable with existing mocks. No external services needed.

### 3.1 Chunk Boundary Improvements

- [ ] **Sentence-aware splitting** — When a paragraph exceeds `ChunkConfig.Size`, split at sentence boundaries (`. `, `? `, `! `) instead of adding the whole paragraph as an oversized chunk. Keep current behaviour as fallback when no sentence boundaries exist.
- [ ] **Overlap boundary alignment** — Current overlap slices by rune count from the end of the previous chunk. Improve by aligning overlap to word boundaries (find the nearest space before the overlap point) to avoid splitting mid-word.
- [ ] **Tests** — (a) Sentence splitting with 3 sentences > Size, (b) overlap word boundary alignment, (c) existing tests still pass (no regression).

### 3.2 Collection Management Helpers

- [ ] **Create `collections.go`** — Helper functions for collection lifecycle:
  - `ListCollections(ctx, store VectorStore) ([]string, error)` — wraps store method
  - `DeleteCollection(ctx, store VectorStore, name string) error` — wraps store method
  - `CollectionStats(ctx, store VectorStore, name string) (*CollectionInfo, error)` — point count, vector size, status. Needs `CollectionInfo` struct (not Qdrant-specific).
- [ ] **Add `ListCollections` and `DeleteCollection` to VectorStore interface** — Currently these methods exist on `QdrantClient` but NOT on the `VectorStore` interface. Add them and update mock.
- [ ] **Tests** — Mock-based tests for all helpers, error injection.

### 3.3 Keyword Pre-Filter

- [ ] **Create `keyword.go`** — `KeywordFilter(results []QueryResult, keywords []string) []QueryResult` — re-ranks results by boosting scores for results containing query keywords. Pure string matching (case-insensitive `strings.Contains`).
  - Boost formula: `score *= 1.0 + 0.1 * matchCount` (each keyword match adds 10% boost)
  - Re-sort by boosted score descending
- [ ] **Add `Keywords bool` to QueryConfig** — When true, extract keywords from query text and apply KeywordFilter after vector search.
- [ ] **Tests** — (a) No keywords (passthrough), (b) single keyword boost, (c) multiple keywords, (d) case insensitive, (e) no matches (scores unchanged).

### 3.4 Benchmarks

- [ ] **Create `benchmark_test.go`** — No build tag (mock-only):
  - `BenchmarkChunk` — 10KB markdown document, default config
  - `BenchmarkChunkWithOverlap` — Same document, overlap=100
  - `BenchmarkQuery_Mock` — Query with mock embedder + mock store
  - `BenchmarkIngest_Mock` — Ingest 10 files with mock embedder + mock store
  - `BenchmarkFormatResults` — FormatResultsText/Context/JSON with 20 results
  - `BenchmarkKeywordFilter` — 100 results, 5 keywords

## Phase 4: GPU Embeddings

- [ ] **ROCm Ollama** — Test Ollama with ROCm on the RX 7800 XT. Measure embedding throughput.
- [ ] **Batch optimisation** — EmbedBatch currently calls Embed sequentially. Ollama may support batch API.
- [ ] **Integration benchmarks** — Live Qdrant + Ollama chunking/embedding/search latency.

---

## Known Issues

1. ~~**go.mod had wrong replace path**~~ — Fixed by Charon.
2. ~~**Qdrant and Ollama not running on snider-linux**~~ — **Resolved.** Qdrant v1.16.3 (Docker) and Ollama with ROCm + nomic-embed-text now running on localhost.
3. ~~**No mocks/interfaces**~~ — **Resolved in Phase 2.** `Embedder` and `VectorStore` interfaces extracted; mock implementations in `mock_test.go`.
4. **`log.E` returns error** — `forge.lthn.ai/core/go/pkg/log.E` wraps errors with component context. This is the framework's logging pattern.

## Platform

- **OS**: Ubuntu (linux/amd64) — snider-linux
- **Co-located with**: go-rocm, go-p2p

## Workflow

1. Charon dispatches tasks here after review
2. Pick up tasks in phase order
3. Mark `[x]` when done, note commit hash
4. New discoveries → add notes, flag in FINDINGS.md
