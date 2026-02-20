# go-rag

Retrieval-Augmented Generation library for Go. Provides document chunking with three-level Markdown splitting and configurable overlap, embedding generation via Ollama, vector storage and cosine-similarity search via Qdrant (gRPC), keyword boosting post-filter, and result formatting in plain text, XML (for LLM prompt injection), or JSON. Designed around `Embedder` and `VectorStore` interfaces that decouple business logic from service implementations and enable mock-based testing.

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
go test ./...                        # unit + mock tests (no external services)
go test -tags rag ./...              # full suite with live Qdrant + Ollama
go test -race ./...
go build ./...
```

## Licence

European Union Public Licence 1.2 — see [LICENCE](LICENCE) for details.
