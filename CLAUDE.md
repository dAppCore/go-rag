# CLAUDE.md

Module: `forge.lthn.ai/core/go-rag`

Retrieval-Augmented Generation — document chunking, Ollama embeddings, Qdrant vector storage and search.

## Commands

```bash
go test ./...                        # Unit + mock tests (no services needed)
go test -tags rag ./...              # Full suite including live Qdrant + Ollama
go test -v -run TestName ./...       # Single test
go test -bench=. -benchmem ./...     # Benchmarks (mock-only)
go test -tags rag -bench=. ./...     # Benchmarks with live services
```

## Architecture

| File | Purpose |
|------|---------|
| `embedder.go` | `Embedder` interface |
| `vectorstore.go` | `VectorStore` interface + `CollectionInfo` |
| `chunk.go` | Markdown chunking — sections, paragraphs, sentences, overlap |
| `ollama.go` | `OllamaClient` — implements `Embedder` |
| `qdrant.go` | `QdrantClient` — implements `VectorStore` |
| `ingest.go` | Ingestion pipeline |
| `query.go` | Query pipeline + result formatting |
| `keyword.go` | Keyword boosting post-filter |
| `collections.go` | Collection management helpers |
| `helpers.go` | Convenience wrappers (`*With` and default-client variants) |

See `docs/architecture.md` for full design detail.

## Key API

```go
// Ingest a directory (interface-accepting variant)
IngestDirWith(ctx, store, embedder, directory, collectionName string, recreate bool) error

// Ingest a single file
IngestFileWith(ctx, store, embedder, filePath, collectionName string) (int, error)

// Query for relevant context
results, err := QueryWith(ctx, store, embedder, question, collectionName string, topK int)
context, err := QueryContextWith(ctx, store, embedder, question, collectionName string, topK int)

// Format results
FormatResultsText(results)    // plain text
FormatResultsContext(results) // XML for LLM injection
FormatResultsJSON(results)    // JSON array
```

## Coding Standards

- UK English (colour, organisation, initialise, behaviour)
- Conventional commits: `type(scope): description`
- Co-Author: `Co-Authored-By: Virgil <virgil@lethean.io>`
- Licence: EUPL-1.2
- Tests: testify assert/require
- Integration tests: `//go:build rag` build tag
- Mocks: `mockEmbedder` and `mockVectorStore` in `mock_test.go`

## Service Defaults

| Service | Host | Port | Notes |
|---------|------|------|-------|
| Qdrant | localhost | 6334 | gRPC |
| Ollama | localhost | 11434 | HTTP |
| Model | — | — | `nomic-embed-text` (768 dims) |
