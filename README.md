[![Go Reference](https://pkg.go.dev/badge/forge.lthn.ai/core/go-rag.svg)](https://pkg.go.dev/forge.lthn.ai/core/go-rag)
[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](LICENSE.md)
[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat&logo=go)](go.mod)

# go-rag

Retrieval-Augmented Generation library for Go. Provides document chunking with three-level Markdown splitting and configurable overlap, embedding generation via Ollama, vector storage and cosine-similarity search via Qdrant (gRPC), keyword boosting post-filter, and result formatting in plain text, XML (for LLM prompt injection), or JSON. Ingestion accepts Markdown, text, PDF, and `.markdown` documents via `ShouldProcess()` / `FileExtensions()`. Designed around `Embedder` and `VectorStore` interfaces that decouple business logic from service implementations and enable mock-based testing.

The package also exposes convenience helpers such as `QueryWith`, `QueryContextWith`, `IngestDirWith`, `IngestFileWith`, `QueryDocs`, `QueryDocsContext`, `IngestDirectory`, `IngestSingleFile`, `CollectionStats`, `ListCollectionsSeq`, `FileExtensions`, `ShouldProcess`, and `JoinResults` for common one-shot and prompt-assembly flows.

**Module**: `forge.lthn.ai/core/go-rag`
**Licence**: EUPL-1.2
**Language**: Go 1.25

## Quick Start

```go
import "forge.lthn.ai/core/go-rag"

// Ingest a directory of Markdown files
err := rag.IngestDirectory(ctx, "/path/to/docs", "my-collection", false)

// Query for relevant context (suitable for LLM prompt injection)
context, err := rag.QueryDocsContext(ctx, "how does rate limiting work?", "my-collection", 5)

// Interface-accepting variants for long-lived processes
results, err := rag.QueryWith(ctx, store, embedder, "question", "collection", 5)
```

## Documentation

- [Architecture](docs/architecture.md) — interfaces, chunking strategy, ingestion pipeline, query pipeline, keyword boosting
- [Development Guide](docs/development.md) — prerequisites, build, test tags, live vs mock tests
- [Project History](docs/history.md) — completed phases and known limitations

## Build & Test

```bash
GOWORK=off go test ./...             # unit + mock tests (no external services)
go test -tags rag ./...              # full suite with live Qdrant + Ollama
GOWORK=off go test -race ./...
GOWORK=off go build ./...
```

## Licence

European Union Public Licence 1.2 — see [LICENCE](LICENCE) for details.
