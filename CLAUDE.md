# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Module: `dappco.re/go/rag`

## Repo layout

All Go sources, module files, and Go-only infra are now under `go/`.

```text
/                                  # Repo root
├── go/
│   ├── go.mod
│   ├── go.sum
│   ├── go.work
│   ├── go.work.sum
│   ├── cmd/
│   ├── internal/
│   ├── tests/
│   ├── tools/
│   ├── *.go
│   ├── README.md      -> symlink to root README.md
│   ├── CLAUDE.md      -> symlink to root CLAUDE.md
│   ├── AGENTS.md      -> symlink to root AGENTS.md
│   └── docs/          -> symlink to root docs/
├── docs/             (RAG specs and docs)
├── specs/            (RAG spec definitions)
├── README.md
├── CLAUDE.md
├── AGENTS.md
├── CONTRIBUTING.md
├── sonar-project.properties
├── LICENSE / LICENCE
└── .woodpecker.yml
```

Retrieval-Augmented Generation library for Go — document chunking, Ollama embeddings, Qdrant vector storage and cosine-similarity search.

## Commands

```bash
cd go && go build ./...                       # Build library + CLI
cd go && go test ./...                        # Unit + mock tests (no services needed)
cd go && go test -tags rag ./...              # Full suite including live Qdrant + Ollama
cd go && go test -v -run TestName ./...       # Single test
cd go && go test -race ./...                  # Race detector
cd go && go test -bench=. -benchmem ./...     # Benchmarks (mock-only)
cd go && go test -tags rag -bench=. ./...     # Benchmarks with live services
cd go && gofmt -w .                           # Format
cd go && go vet ./...                         # Vet
cd go && golangci-lint run ./...              # Lint
```

## Architecture

The library is built around two core interfaces (`Embedder` in `go/embedder.go`, `VectorStore` in `go/vectorstore.go`) that decouple business logic from service implementations. All pipeline code operates against these interfaces; concrete clients (`OllamaClient`, `QdrantClient`) and test mocks (`mockEmbedder`, `mockVectorStore` in `go/mock_test.go`) satisfy them.

**Two pipeline flows** drive everything:

1. **Ingestion** (`go/ingest.go`): directory walk -> `ChunkMarkdown` (three-level split in `go/chunk.go`) -> embed per chunk -> batch upsert to vector store
2. **Query** (`go/query.go`): embed question -> vector search -> threshold filter -> optional keyword boosting (`go/keyword.go`) -> format results (text/XML/JSON)

**Two-tier helper pattern** in `go/helpers.go`:
- `*With` variants (e.g. `QueryWith`, `IngestDirWith`) accept pre-constructed `VectorStore`+`Embedder` — use for long-lived processes
- Default-client wrappers (e.g. `QueryDocs`, `IngestDirectory`) create fresh connections per call with health checks — use for CLI/one-shot operations

**CLI** (`go/cmd/rag/`): CLI subcommands registered via `AddRAGSubcommands()`, mounted under `core ai rag`.

## Coding Standards

- UK English (colour, organisation, initialise, behaviour)
- Conventional commits: `type(scope): description`
- Co-Author: `Co-Authored-By: Virgil <virgil@lethean.io>`
- Licence: EUPL-1.2
- Tests: stdlib `testing` with the local helpers in `go/test_helpers_test.go`; do not add testify
- Integration tests: `//go:build rag` build tag — requires live Qdrant + Ollama
- Mocks: `mockEmbedder` and `mockVectorStore` in `go/mock_test.go` — error injection via fields, call tracking for verification

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
