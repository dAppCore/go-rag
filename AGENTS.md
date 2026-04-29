# Agent Guide

This repository is the RAG package for the `dappco.re/go` ecosystem. It
provides Markdown/PDF ingestion, keyword fallback search, Ollama embeddings,
Qdrant vector storage, and the `rag` CLI subcommands used by downstream tools.

The public package is rooted at `dappco.re/go/rag`. Keep production code in
the source file that owns the exported symbol, keep the sibling test in the
matching `<file>_test.go`, and keep runnable examples in
`<file>_example_test.go`. The compliance audit is intentionally file-aware:
tests for `query.go` belong in `query_test.go`, not in a shared compliance
file.

Use `dappco.re/go` wrappers for formatting, JSON, strings, paths, filesystem,
process, and test assertions. Direct imports of stdlib packages covered by
core wrappers are not accepted in this repository, including in tests and CLI
fixtures. Imports that do not have core equivalents, such as `context`,
`net/http`, `net/url`, `io`, `math`, and third-party client packages, remain
normal Go imports.

Local tests rely on the in-memory mocks in `mock_test.go`; prefer those mocks
over networked Ollama or Qdrant services unless the test is explicitly marked
as an integration test. CLI compatibility code lives under
`internal/compat/cli` so the command package can build against the current
core/go reference while preserving the small command surface this repository
needs.

Before handing work back, run the repository gate from `BRIEF.md`: tidy, vet,
test, `gofmt -l .`, and the v0.9.0 audit script. The audit must report
`verdict: COMPLIANT` with every counter at zero.
