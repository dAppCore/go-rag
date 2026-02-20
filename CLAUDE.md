# CLAUDE.md

## What This Is

Retrieval-Augmented Generation with vector search. Module: `forge.lthn.ai/core/go-rag`

Provides document chunking, embedding via Ollama, vector storage/search via Qdrant, and formatted context retrieval for AI prompts.

## Commands

```bash
go test ./...                    # Run all tests
go test -v -run TestChunk        # Single test
```

## Architecture

| File | Purpose |
|------|---------|
| `chunk.go` | Document chunking — splits markdown/text into semantic chunks |
| `embedder.go` | `Embedder` interface — abstraction for embedding providers |
| `vectorstore.go` | `VectorStore` interface — abstraction for vector storage backends |
| `ingest.go` | Ingestion pipeline — reads files, chunks, embeds, stores (accepts interfaces) |
| `query.go` | Query interface — search vectors, format results as text/JSON/XML (accepts interfaces) |
| `qdrant.go` | Qdrant vector DB client — implements `VectorStore` |
| `ollama.go` | Ollama embedding client — implements `Embedder` |
| `helpers.go` | Convenience wrappers — `*With` variants accept interfaces, defaults construct live clients |

## Dependencies

- `forge.lthn.ai/core/go` — Logging (pkg/log)
- `github.com/ollama/ollama` — Embedding API client
- `github.com/qdrant/go-client` — Vector DB gRPC client
- `github.com/stretchr/testify` — Tests

## Key API

```go
// Ingest documents
rag.IngestFile(ctx, cfg, "/path/to/doc.md")
rag.Ingest(ctx, cfg, reader, "source-name")

// Query for relevant context
results, err := rag.Query(ctx, cfg, "search query")
context := rag.FormatResults(results, "text") // or "json", "xml"
```

## Coding Standards

- UK English
- Tests: testify assert/require
- Conventional commits
- Co-Author: `Co-Authored-By: Virgil <virgil@lethean.io>`
- Licence: EUPL-1.2
