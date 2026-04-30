<!-- SPDX-License-Identifier: EUPL-1.2 -->



> Retrieval-augmented generation — chunking, embeddings, qdrant + ollama

[![CI](https://github.com/dappcore/go-rag/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/dappcore/go-rag/actions/workflows/ci.yml)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=alert_status)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Coverage](https://codecov.io/gh/dappcore/go-rag/branch/dev/graph/badge.svg)](https://codecov.io/gh/dappcore/go-rag)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=security_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=code_smells)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=dappcore_go-rag&metric=ncloc)](https://sonarcloud.io/dashboard?id=dappcore_go-rag)
[![Go Reference](https://pkg.go.dev/badge/dappco.re/go/go-rag.svg)](https://pkg.go.dev/dappco.re/go/go-rag)
[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](https://eupl.eu/1.2/en/)


Retrieval-Augmented Generation library for Go. Provides document chunking with three-level Markdown splitting plus sentence- and paragraph-based chunkers, configurable overlap, embedding generation via Ollama, vector storage and cosine-similarity search via Qdrant (gRPC), TF-IDF keyword fallback and keyword boosting post-filter, and result formatting in plain text, XML (for LLM prompt injection), or JSON. Ingestion accepts Markdown, text, PDF, and `.markdown` documents via `ShouldProcess()` / `FileExtensions()`. Designed around `Embedder` and `VectorStore` interfaces that decouple business logic from service implementations and enable mock-based testing.

The package also exposes convenience helpers such as `QueryWith`, `QueryContextWith`, `IngestDirWith`, `IngestFileWith`, `QueryDocs`, `QueryDocsContext`, `IngestDirectory`, `IngestSingleFile`, `CollectionStats`, `ListCollectionsSeq`, `FileExtensions`, `ShouldProcess`, `JoinResults`, `KeywordFilterSeq`, `ChunkBySentences`, and `ChunkByParagraphs` for common one-shot and prompt-assembly flows.

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
GOWORK=off go test -tags rag ./...   # full suite with live Qdrant + Ollama
GOWORK=off go test -race ./...
GOWORK=off go build ./...
```

## Licence

European Union Public Licence 1.2 — see [LICENCE](LICENCE) for details.
