# go-rag Project History

## Origin

go-rag was extracted from `forge.lthn.ai/core/go-ai` on 19 February 2026 by Virgil.

The source code lived in `go-ai/rag/` as a subsystem of the meta AI hub. It had zero internal dependencies on go-ai's other packages (model management, MCP tools, inference backends), making extraction straightforward. At the time of extraction the package comprised 7 Go files (~1,017 LOC excluding tests) and a single test file covering only `chunk.go`.

**go-ai consumers retained**:
- `go-ai/ai/rag.go` — wraps go-rag as `QueryRAGForTask()`, a facade used by AI task runners.
- `go-ai/mcp/tools_rag.go` — exposes go-rag operations as MCP tool handlers (`rag_query`, `rag_ingest`, `rag_collections`).

These consumers were updated to import `forge.lthn.ai/core/go-rag` instead of the internal path.

## Phase 0: Environment Setup (19 February 2026, Charon)

**Problem**: The `go.mod` replace directive pointed to `../core` (the root Go workspace module) rather than `../go` (the `go` sibling package providing logging). Tests failed to compile.

**Resolution**: Replace directive corrected to `../go`. Qdrant v1.16.3 started via Docker. Ollama with ROCm installed natively on snider-linux (AMD RX 7800 XT, gfx1100). Model `nomic-embed-text` (F16, 274MB) pulled. All 32 initial tests passed after the fix.

## Phase 1: Pure-Function Tests (19–20 February 2026)

**Commit**: `acb987a`

Coverage before: 18.4% (8 tests in `chunk_test.go` only).
Coverage after: 38.8% (66 tests across 4 test files).

Targeted all functions that did not require live external services:

- `FormatResultsText`, `FormatResultsContext`, `FormatResultsJSON`
- All `Default*Config` functions
- `OllamaClient.EmbedDimension` (pure switch on model name)
- `OllamaClient.Model`
- `QdrantClient.valueToGo` (protobuf value conversion)
- `ChunkID`, `ChunkMarkdown` (extended edge cases: empty input, unicode, headers-only, long paragraphs)
- `pointIDToString`

**Discovery**: `OllamaClient` can be constructed with a nil `client` field for testing pure methods. The remaining ~61% untested code was entirely in functions requiring live services.

**Discovery**: `pointIDToString` has an unreachable default branch — Qdrant only exposes `NewIDNum` and `NewIDUUID` constructors, so the third `PointIdOptions` case cannot be constructed without reflection. Coverage of 83.3% is the practical maximum for that function.

## Phase 2: Interface Extraction and Mock Infrastructure (20 February 2026)

**Commit**: `a49761b`

Coverage before: 38.8%.
Coverage after: 69.0% (135 tests across 7 test files).

Two interfaces were extracted to decouple business logic from concrete service clients:

- `Embedder` (embedder.go) — `Embed`, `EmbedBatch`, `EmbedDimension`
- `VectorStore` (vectorstore.go) — `CreateCollection`, `CollectionExists`, `DeleteCollection`, `ListCollections`, `CollectionInfo`, `UpsertPoints`, `Search`

`Ingest`, `IngestFile`, and `Query` were updated to accept interfaces rather than concrete `*QdrantClient` and `*OllamaClient` parameters. These changes were backwards-compatible because the concrete types satisfy the interfaces.

`*With` helper variants were added (`QueryWith`, `QueryContextWith`, `IngestDirWith`, `IngestFileWith`) to accept pre-constructed interfaces. The existing convenience wrappers (`QueryDocs`, `IngestDirectory`, `IngestSingleFile`) were refactored to delegate to the `*With` variants.

`mock_test.go` was created containing `mockEmbedder` (deterministic 0.1 vectors, call tracking, error injection) and `mockVectorStore` (in-memory map, fake descending scores, filter support, call tracking).

69 new mock-based tests were written: 23 for `Ingest`/`IngestFile`, 12 for `Query`, 16 for helpers, plus updated chunk tests.

**Discovery**: Interface method signatures must match exactly. `EmbedDimension` returns `uint64` (not `int`), and `Search` takes `limit uint64` and `filter map[string]string`. The task specification suggested approximate signatures; the source was authoritative.

## Phase 3: Integration Tests with Live Services (20 February 2026)

**Commit**: `e90f281`

Coverage before: 69.0%.
Coverage after: 89.2% (204 tests across 10 test files).

Three integration test files were added under the `//go:build rag` build tag:

- `qdrant_integration_test.go` — 11 subtests: health check, collection create/delete/list/info, exists check, upsert and search, payload filter, empty upsert, ID validation, overwrite behaviour.
- `ollama_integration_test.go` — 9 subtests: model verification, single embed, batch embed, embedding determinism, dimension match, model name, different texts producing different vectors, non-zero values, empty string handling.
- `integration_test.go` — 12 subtests: full ingest+query pipeline, format results in all three formats, `IngestFile`, `QueryWith`, `QueryContextWith`, `IngestDirWith`, `IngestFileWith`, `QueryDocs`, `IngestDirectory`, recreate flag, semantic similarity verification.

**Discovery**: Qdrant point IDs must be valid UUIDs. Arbitrary strings such as `"point-alpha"` are rejected. The `ChunkID` MD5 hex output (32 lowercase hex chars) is accepted by Qdrant's UUID parser.

**Discovery**: Qdrant indexing requires a brief delay (500ms sleep) between upsert and search in tests to avoid race conditions on slower hardware.

**Discovery**: Semantic similarity works as expected. Queries about Go programming rank programming documents above cooking documents. Cosine distance combined with `nomic-embed-text` provides meaningful semantic differentiation.

**Discovery**: `QueryDocs` and `IngestDirectory` open a new gRPC connection on each call. For high-throughput use the `*With` variants with a shared client are preferable.

## Phase 3: Enhancements (20 February 2026)

**Commit**: `cf26e88`

### Sentence-Aware Splitting

`ChunkMarkdown` was enhanced to split oversized paragraphs at sentence boundaries (`. `, `? `, `! `) rather than adding the whole paragraph as an oversized chunk. The original fallback (adding the paragraph as-is) is retained when no sentence boundaries exist.

### Word-Boundary Overlap Alignment

The overlap logic was improved to align the overlap start point to the nearest word boundary within the overlap slice, avoiding split words at chunk beginnings. Rune-safe slicing was retained.

### Collection Management Helpers

`collections.go` was created with three package-level helpers: `ListCollections`, `DeleteCollection`, `CollectionStats`. `ListCollections` and `DeleteCollection` were added to the `VectorStore` interface; `mockVectorStore` was updated accordingly.

### Keyword Boosting

`keyword.go` was created with `KeywordFilter` (score boosting: +10% per matching keyword) and `extractKeywords` (words of 3+ characters). `QueryConfig` gained a `Keywords bool` field; `Query` applies the filter when the field is true.

### Benchmarks

`benchmark_test.go` was created (no build tag, mock-only): `BenchmarkChunk`, `BenchmarkChunkWithOverlap`, `BenchmarkQuery_Mock`, `BenchmarkIngest_Mock`, `BenchmarkFormatResults` (all three formats), `BenchmarkKeywordFilter`.

## Phase 4: GPU Benchmarks (20 February 2026, Charon)

Service benchmarks were added in `benchmark_gpu_test.go` under the `//go:build rag` tag.

**Hardware**: AMD Ryzen 9 9950X, AMD RX 7800 XT (ROCm, gfx1100), Qdrant v1.16.3 (Docker).

**Key findings**:

- Single embed latency: 10.3ms (97/sec). Text length (50 to 2000 chars) has negligible effect.
- Batch embed throughput equals single embed throughput — there is no batch API in Ollama.
- Qdrant search: 111µs for 100 points (9,042 QPS), 152µs for 200 points.
- Pipeline bottleneck: embedding accounts for ~95% of a full ingest+query cycle.
- GPU compute per embed: ~2ms. The remaining ~8ms is HTTP and serialisation overhead on localhost.

## Known Limitations

**No batch embedding API**: Ollama's `/api/embed` endpoint accepts a single `input` string. `EmbedBatch` is implemented as a sequential loop. Parallel embedding requires multiple concurrent HTTP clients, which is left to callers.

**Convenience wrappers open connections per call**: `QueryDocs`, `IngestDirectory`, and `IngestSingleFile` construct and close a new gRPC connection on every invocation. This is acceptable for CLI tooling but unsuitable for server processes or tight loops.

**Category detection is path-based**: `Category()` classifies files by keyword matching on the file path. This is a heuristic with no configuration mechanism. False positives and missed classifications are expected for non-standard directory structures.

**Qdrant version mismatch warning**: The `qdrant/go-client` v1.16.2 library logs `WARN Unable to compare versions` when connecting to Qdrant v1.16.3. The warning is cosmetic; all operations function correctly.

**`filepath.Rel` error branch**: The error branch in `Ingest` for `filepath.Rel` failures (incrementing `stats.Errors` and continuing) is not covered by tests. This path requires a filesystem or OS-level anomaly to trigger and is not reachable via normal input.

**EmbedDimension default fallback**: Unknown model names return 768 (the `nomic-embed-text` dimension). If a different model is configured and its dimension is unknown to the library, the collection will be created with an incorrect vector size, causing upsert failures at the Qdrant level.

## Future Considerations

- **Concurrent embedding**: A configurable worker pool for parallel `Embed` calls during batch ingestion would reduce pipeline latency in proportion to the number of workers, up to Ollama's concurrency limit.
- **Streaming results**: `Query` returns all results after the full search completes. A channel-based API could stream results as they pass the threshold filter.
- **Additional backends**: The `VectorStore` interface is ready for alternative implementations (Weaviate, pgvector, Milvus). The `Embedder` interface supports OpenAI, Cohere, or any HTTP embedding API.
- **Configurable chunking strategies**: The current chunker is specialised for Markdown. Plain text, HTML, or code files would benefit from different splitting strategies selectable via a strategy interface.
- **Metadata filtering**: The current filter mechanism supports exact-match on a single field (`category`). Range queries and multi-field filters would require extending the `VectorStore` interface or passing an opaque filter type.
