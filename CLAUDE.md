# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Module: `forge.lthn.ai/core/go-rag`

Retrieval-Augmented Generation library for Go — document chunking, Ollama embeddings, Qdrant vector storage and cosine-similarity search.

## Commands

```bash
go build ./...                       # Build library + CLI
go test ./...                        # Unit + mock tests (no services needed)
go test -tags rag ./...              # Full suite including live Qdrant + Ollama
go test -v -run TestName ./...       # Single test
go test -race ./...                  # Race detector
go test -bench=. -benchmem ./...     # Benchmarks (mock-only)
go test -tags rag -bench=. ./...     # Benchmarks with live services
gofmt -w .                           # Format
go vet ./...                         # Vet
golangci-lint run ./...              # Lint
```

## Architecture

The library is built around two core interfaces (`Embedder` in `embedder.go`, `VectorStore` in `vectorstore.go`) that decouple business logic from service implementations. All pipeline code operates against these interfaces; concrete clients (`OllamaClient`, `QdrantClient`) and test mocks (`mockEmbedder`, `mockVectorStore` in `mock_test.go`) satisfy them.

**Two pipeline flows** drive everything — see `docs/architecture.md` for the full data-flow diagrams:

1. **Ingestion** (`ingest.go`): directory walk -> `ChunkMarkdown` (three-level split in `chunk.go`) -> embed per chunk -> batch upsert to vector store
2. **Query** (`query.go`): embed question -> vector search -> threshold filter -> optional keyword boosting (`keyword.go`) -> format results (text/XML/JSON)

**Two-tier helper pattern** in `helpers.go`:
- `*With` variants (e.g. `QueryWith`, `IngestDirWith`) accept pre-constructed `VectorStore`+`Embedder` — use for long-lived processes
- Default-client wrappers (e.g. `QueryDocs`, `IngestDirectory`) create fresh connections per call with health checks — use for CLI/one-shot operations

**CLI** (`cmd/rag/`): cobra subcommands registered via `AddRAGSubcommands()`, mounted under `core ai rag`. Uses `forge.lthn.ai/core/cli` and `forge.lthn.ai/core/go-i18n` for i18n'd flag descriptions.

## Coding Standards

- UK English (colour, organisation, initialise, behaviour)
- Conventional commits: `type(scope): description`
- Co-Author: `Co-Authored-By: Virgil <virgil@lethean.io>`
- Licence: EUPL-1.2
- Tests: testify assert/require
- Integration tests: `//go:build rag` build tag — requires live Qdrant + Ollama
- Mocks: `mockEmbedder` and `mockVectorStore` in `mock_test.go` — error injection via fields, call tracking for verification

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `QDRANT_HOST` | `localhost` | Qdrant server host |
| `QDRANT_PORT` | `6334` | Qdrant gRPC port |
| `OLLAMA_HOST` | `localhost` | Ollama server host |
| `OLLAMA_PORT` | `11434` | Ollama HTTP port |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model name (768 dims) |

## Service Defaults

| Service | Host | Port | Protocol |
|---------|------|------|----------|
| Qdrant | localhost | 6334 | gRPC |
| Ollama | localhost | 11434 | HTTP |
