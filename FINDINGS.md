# FINDINGS.md — go-rag Research & Discovery

## 2026-02-19: Split from go-ai (Virgil)

### Origin

Extracted from `forge.lthn.ai/core/go-ai/rag/`. Zero internal go-ai dependencies.

### What Was Extracted

- 7 Go files (~1,017 LOC excluding tests)
- 1 test file (chunk_test.go)

### Key Finding: Minimal Test Coverage

Only chunk.go has tests. The Qdrant and Ollama clients are untested — they depend on external services (Qdrant server, Ollama API) which makes unit testing harder. Consider mock interfaces.

### Consumers

- `go-ai/ai/rag.go` wraps this as `QueryRAGForTask()` facade
- `go-ai/mcp/tools_rag.go` exposes RAG as MCP tools
