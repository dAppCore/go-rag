# TODO.md — go-rag Task Queue

## Phase 1: Post-Split

- [ ] **Verify tests pass standalone** — Run `go test ./...`. Confirm chunk_test.go passes.
- [ ] **Add missing tests** — Only chunk.go has tests. Need: query, ingest, qdrant client, ollama client.
- [ ] **Benchmark chunking** — No perf baselines. Add BenchmarkChunk for various document sizes.

## Phase 2: Enhancements

- [ ] **Embedding model selection** — Currently hardcoded to Ollama. Add backend interface for alternative embedding providers.
- [ ] **Collection management** — Add list/delete collection operations for Qdrant.
- [ ] **Hybrid search** — Combine vector similarity with keyword matching for better recall.
- [ ] **Chunk overlap** — Current chunking may lose context at boundaries. Add configurable overlap.

---

## Workflow

1. Virgil in core/go writes tasks here after research
2. This repo's session picks up tasks in phase order
3. Mark `[x]` when done, note commit hash
