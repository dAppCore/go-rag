---
title: go-rag
description: Retrieval-Augmented Generation library for Go — document chunking, Ollama embeddings, Qdrant vector storage, and formatted context retrieval for LLM prompt injection.
---

# go-rag

`forge.lthn.ai/core/go-rag` is a Retrieval-Augmented Generation library for Go. It handles the full RAG pipeline: splitting documents into chunks, generating embeddings via Ollama, storing and searching vectors in Qdrant (gRPC), applying keyword boosting, and formatting results for human display or LLM prompt injection.

The library is built around two core interfaces -- `Embedder` and `VectorStore` -- that decouple business logic from service implementations. You can swap backends, inject mocks for testing, or run the full pipeline against live services with the same API.

**Module**: `forge.lthn.ai/core/go-rag`
**Go version**: 1.26
**Licence**: EUPL-1.2

## Quick Start

```go
import "forge.lthn.ai/core/go-rag"

// Ingest a directory of Markdown files into a Qdrant collection
err := rag.IngestDirectory(ctx, "/path/to/docs", "my-collection", false)

// Query for relevant context (XML format, suitable for LLM prompt injection)
context, err := rag.QueryDocsContext(ctx, "how does rate limiting work?", "my-collection", 5)

// For long-lived processes, construct clients once and use the *With variants
qdrantClient, _ := rag.NewQdrantClient(rag.DefaultQdrantConfig())
ollamaClient, _ := rag.NewOllamaClient(rag.DefaultOllamaConfig())

results, err := rag.QueryWith(ctx, qdrantClient, ollamaClient, "question", "collection", 5)
```

The convenience wrappers (`IngestDirectory`, `QueryDocs`, etc.) create new connections on each call, which is fine for CLI usage. For server processes or loops, use the `*With` variants with pre-constructed clients to avoid per-call connection overhead.

## Package Layout

| File | Purpose |
|------|---------|
| `embedder.go` | `Embedder` interface -- `Embed`, `EmbedBatch`, `EmbedDimension` |
| `vectorstore.go` | `VectorStore` interface -- collection management, upsert, search |
| `chunk.go` | Markdown chunking with three-level splitting (sections, paragraphs, sentences) and configurable overlap |
| `ollama.go` | `OllamaClient` -- implements `Embedder` via the Ollama HTTP API |
| `qdrant.go` | `QdrantClient` -- implements `VectorStore` via the Qdrant gRPC API |
| `ingest.go` | Ingestion pipeline -- walk directory, chunk files, embed, batch upsert |
| `query.go` | Query pipeline -- embed query, vector search, threshold filter, format results |
| `keyword.go` | Keyword boosting post-filter for re-ranking search results |
| `collections.go` | Package-level collection management helpers |
| `helpers.go` | Convenience wrappers -- `*With` variants and default-client functions |
| `cmd/rag/` | CLI subcommands (`ingest`, `query`, `collections`) for the `core` binary |

## CLI Commands

The package provides CLI subcommands mounted under `core ai rag`:

```bash
# Ingest a directory of Markdown files
core ai rag ingest /path/to/docs --collection my-docs --recreate

# Query the vector database
core ai rag query "how does the module system work?" --top 10 --format context

# List and manage collections
core ai rag collections --stats
core ai rag collections --delete old-collection
```

All commands accept `--qdrant-host`, `--qdrant-port`, `--ollama-host`, `--ollama-port`, and `--model` flags, with defaults overridable via environment variables (`QDRANT_HOST`, `QDRANT_PORT`, `OLLAMA_HOST`, `OLLAMA_PORT`, `EMBEDDING_MODEL`).

## Dependencies

| Dependency | Role |
|------------|------|
| `forge.lthn.ai/core/go-log` | Structured error wrapping (`log.E`) |
| `forge.lthn.ai/core/go-i18n` | Internationalised CLI strings |
| `forge.lthn.ai/core/cli` | CLI framework (cobra-based commands) |
| `github.com/ollama/ollama` | Ollama HTTP client for embedding generation |
| `github.com/qdrant/go-client` | Qdrant gRPC client for vector storage and search |
| `github.com/stretchr/testify` | Test assertions (test-only) |

Transitive dependencies include `google.golang.org/grpc`, `google.golang.org/protobuf`, and `github.com/google/uuid`.

## Service Defaults

| Service | Host | Port | Protocol |
|---------|------|------|----------|
| Qdrant | localhost | 6334 | gRPC |
| Ollama | localhost | 11434 | HTTP |

The default embedding model is `nomic-embed-text` (768 dimensions). Other supported models include `mxbai-embed-large` (1024 dimensions) and `all-minilm` (384 dimensions).

## Further Reading

- [Architecture](architecture.md) -- interfaces, chunking strategy, ingestion pipeline, query pipeline, keyword boosting, performance characteristics
- [Development](development.md) -- prerequisites, build commands, test patterns, coding standards, contribution guidelines
