# TODO.md — go-rag Task Queue

Dispatched from core/go orchestration. Pick up tasks in phase order.

---

## Phase 0: Environment Setup

- [ ] **Fix go.mod replace directive** — Was `../core`, corrected to `../go`. Commit and push. (Charon, 19 Feb 2026)
- [ ] **Run Qdrant locally** — Docker: `docker run -d -p 6333:6333 -p 6334:6334 qdrant/qdrant`. Test with `curl http://localhost:6334/healthz`.
- [ ] **Install Ollama** — `curl -fsSL https://ollama.com/install.sh | sh`. Pull embedding model: `ollama pull nomic-embed-text`.
- [ ] **Verify both services** — `go test -v -run TestQdrant ./...` and `go test -v -run TestOllama ./...` (write these tests in Phase 1).

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

- [ ] **Chunk overlap** — Configurable overlap already in ChunkConfig but needs better boundary handling.
- [ ] **Collection management** — List/delete collection operations (already in qdrant.go but untested).
- [ ] **Hybrid search** — Combine vector similarity with keyword matching.
- [ ] **Embedding model selection** — Backend interface for alternative providers (not just Ollama).

## Phase 4: GPU Embeddings

- [ ] **ROCm Ollama** — Test Ollama with ROCm on the RX 7800 XT. Measure embedding throughput.
- [ ] **Batch optimisation** — EmbedBatch currently calls Embed sequentially. Ollama may support batch API.
- [ ] **Benchmarks** — Chunking speed, embedding throughput, search latency.

---

## Known Issues

1. **go.mod had wrong replace path** — `../core` should be `../go`. Fixed by Charon.
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
