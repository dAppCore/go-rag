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

- [ ] **Qdrant client tests** — Create collection, upsert, search, delete. Skip if Qdrant unavailable.
- [ ] **Ollama client tests** — Embed single text, embed batch, verify model. Skip if Ollama unavailable.
- [ ] **Query integration test** — Ingest a test doc, query it, verify results.

## Phase 2: Test Infrastructure

- [ ] **Interface extraction** — Extract `Embedder` interface (Embed, EmbedBatch, EmbedDimension) and `VectorStore` interface (Search, Upsert, etc.) so Qdrant and Ollama can be mocked in unit tests.
- [ ] **Mock embedder** — Returns deterministic vectors for testing query/ingest without Ollama.
- [ ] **Mock vector store** — In-memory implementation for testing ingest/query without Qdrant.
- [ ] **Re-test with mocks** — Rewrite ingest + query tests using mock interfaces for fast, reliable CI.

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
2. **Qdrant and Ollama not running on snider-linux** — Need docker setup for Qdrant, native install for Ollama.
3. **No mocks/interfaces** — All external calls go directly to Qdrant/Ollama clients. Unit tests require live services until interfaces are extracted.
4. **`log.E` returns error** — `forge.lthn.ai/core/go/pkg/log.E` wraps errors with component context. This is the framework's logging pattern.

## Platform

- **OS**: Ubuntu (linux/amd64) — snider-linux
- **Co-located with**: go-rocm, go-p2p

## Workflow

1. Charon dispatches tasks here after review
2. Pick up tasks in phase order
3. Mark `[x]` when done, note commit hash
4. New discoveries → add notes, flag in FINDINGS.md
