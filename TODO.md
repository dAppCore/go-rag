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

---

## Linux Homelab Assignment (Virgil, 19 Feb 2026)

This package is assigned to the Linux homelab agent. The homelab can run Qdrant and Ollama natively, giving the dedicated Claude real vector DB and embedding infrastructure to test against.

### Linux-Specific Tasks

- [ ] **Run Qdrant locally** — Docker or native binary on the homelab. Test against real Qdrant instance, not mocks.
- [ ] **Run Ollama locally** — Install Ollama on Linux. Test embedding generation with real models (nomic-embed-text, etc.).
- [ ] **Full pipeline integration test** — Ingest → chunk → embed → store → query end-to-end with real Qdrant + Ollama.
- [ ] **AMD GPU embeddings** — Ollama supports ROCm. Test embedding generation on the RX 7800 XT for faster processing.

### Platform

- **OS**: Ubuntu 24.04 (linux/amd64)
- **GPU**: AMD RX 7800 XT (ROCm, for Ollama GPU acceleration)
- **Co-located with**: go-rocm (AMD GPU inference), go-p2p (networking)

## Workflow

1. Virgil in core/go writes tasks here after research
2. This repo's session picks up tasks in phase order
3. Mark `[x]` when done, note commit hash
