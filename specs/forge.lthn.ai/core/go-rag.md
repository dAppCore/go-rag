# rag
**Import:** `forge.lthn.ai/core/go-rag`
**Files:** 10

## Types

### `ChunkConfig`
`type ChunkConfig struct { Size int; Overlap int }`

Chunking configuration for markdown splitting. `Size` is the target chunk length in characters. `Overlap` is the number of trailing characters from the previous chunk that can be prepended to the next chunk. `ChunkMarkdownSeq` normalizes non-positive sizes to `500` and overlap values outside `0 <= overlap < size` to `0`.

### `Chunk`
`type Chunk struct { Text string; Section string; Index int }`

A chunk emitted from markdown input. `Text` is the chunk body, `Section` is the heading text derived from the enclosing markdown heading, and `Index` is the zero-based chunk number assigned during iteration.

### `CollectionInfo`
`type CollectionInfo struct { Name string; PointCount uint64; VectorSize uint64; Status string }`

Backend-agnostic collection metadata. `QdrantClient.CollectionInfo` fills `PointCount` from Qdrant, derives `VectorSize` from the collection vector config, and maps backend status to `green`, `yellow`, `red`, or `unknown`.

### `Embedder`
`type Embedder interface { Embed(context.Context, string) ([]float32, error); EmbedBatch(context.Context, []string) ([][]float32, error); EmbedDimension() uint64 }`

Interface for embedding providers. `Embed` returns one vector for one string, `EmbedBatch` returns one vector per input string, and `EmbedDimension` reports the vector length used when collections are created.

### `IngestConfig`
`type IngestConfig struct { Directory string; Collection string; Recreate bool; Verbose bool; BatchSize int; Chunk ChunkConfig }`

Directory ingestion settings. `Directory` selects the scan root and falls back to `.` when empty. `Collection` is the target collection name. `Recreate` deletes and recreates an existing collection before ingest. `Verbose` enables per-error logging inside `Ingest`. `BatchSize` controls the number of points per `UpsertPoints` call and falls back to `100` when non-positive. `Chunk` is passed to `ChunkMarkdown`.

### `IngestProgress`
`type IngestProgress func(file string, chunks int, total int)`

Optional progress callback for `Ingest`. It is invoked once per processed file with the file's relative path, the cumulative number of chunks created so far, and the total number of matching files discovered under the scan root.

### `IngestStats`
`type IngestStats struct { Files int; Chunks int; Errors int }`

Summary returned by `Ingest`. `Files` counts processed non-empty files, `Chunks` counts successfully embedded chunks queued for upsert, and `Errors` counts unreadable files and embedding failures encountered during the ingest pass.

### `OllamaConfig`
`type OllamaConfig struct { Host string; Port int; Model string }`

Connection settings for Ollama. `Host` and `Port` define the HTTP endpoint used by `NewOllamaClient`. `Model` selects the embedding model used by all `OllamaClient` embedding calls.

### `OllamaClient`
`type OllamaClient struct { /* unexported fields */ }`

Embedding client backed by `github.com/ollama/ollama/api`. The concrete value stores the configured `api.Client` and the chosen `OllamaConfig`, and it satisfies the `Embedder` interface.

### `Point`
`type Point struct { ID string; Vector []float32; Payload map[string]any }`

Vector point written to a `VectorStore`. `ID` is the point identifier, `Vector` is the embedding payload sent to the store, and `Payload` holds arbitrary metadata fields such as document text, source path, section, category, and chunk index.

### `QdrantConfig`
`type QdrantConfig struct { Host string; Port int; APIKey string; UseTLS bool }`

Connection settings for Qdrant. `Host` and `Port` identify the server, `APIKey` is passed through to the client configuration, and `UseTLS` toggles TLS for the generated connection.

### `QdrantClient`
`type QdrantClient struct { /* unexported fields */ }`

Concrete `VectorStore` implementation backed by `github.com/qdrant/go-client/qdrant`. The value wraps an initialized Qdrant client plus the configuration used to create it.

### `QueryConfig`
`type QueryConfig struct { Collection string; Limit uint64; Threshold float32; Category string; Keywords bool }`

Query settings. `Collection` selects the collection to search. `Limit` is forwarded to the vector store search. `Threshold` filters out results below the given score. `Category` enables an exact payload filter on `category`. `Keywords` enables keyword extraction from the query text and score boosting through `KeywordFilter`.

### `QueryResult`
`type QueryResult struct { Text string; Source string; Section string; Category string; ChunkIndex int; Score float32 }`

Normalized result returned by `Query` and `QuerySeq`. The fields are projected from vector store payloads plus the similarity score returned by the search backend.

### `SearchResult`
`type SearchResult struct { ID string; Score float32; Payload map[string]any }`

Low-level vector search hit. `QdrantClient.Search` fills `ID` from the point identifier, `Score` from the backend similarity score, and `Payload` from the stored point payload converted to native Go values.

### `VectorStore`
`type VectorStore interface { CreateCollection(context.Context, string, uint64) error; CollectionExists(context.Context, string) (bool, error); DeleteCollection(context.Context, string) error; ListCollections(context.Context) ([]string, error); CollectionInfo(context.Context, string) (*CollectionInfo, error); UpsertPoints(context.Context, string, []Point) error; Search(context.Context, string, []float32, uint64, map[string]string) ([]SearchResult, error) }`

Interface for vector backends. `CreateCollection` provisions a collection with a given vector size. `CollectionExists`, `DeleteCollection`, `ListCollections`, and `CollectionInfo` manage collection lifecycle and metadata. `UpsertPoints` writes batches of points. `Search` performs similarity search with an optional payload filter map.

## Functions

### Package Functions

### `func DefaultChunkConfig() ChunkConfig`
Returns the default chunker settings: `Size: 500` and `Overlap: 50`.

### `func ChunkMarkdown(text string, cfg ChunkConfig) []Chunk`
Collects all chunks yielded by `ChunkMarkdownSeq`. Chunking splits the document by `## ` headings, trims empty sections and paragraphs, derives section titles from heading lines, yields small sections intact, and falls back to sentence-based splitting when a paragraph exceeds the configured size.

### `func ChunkMarkdownSeq(text string, cfg ChunkConfig) iter.Seq[Chunk]`
Iterator form of the markdown chunker. Chunks receive sequential `Index` values across the full document. When a chunk would exceed `cfg.Size`, the next chunk starts with word-boundary-aligned overlap text copied from the previous chunk before appending the new paragraph content.

### `func Category(path string) string`
Classifies a lowercased path using fixed substring rules: `ui-component` for `flux` or `ui/component`, `brand` for `brand` or `mascot`, `product-brief` for `brief`, `help-doc` for `help` or `draft`, `task` for `task` or `plan`, `architecture` for `architecture` or `migration`, and `documentation` otherwise.

### `func ChunkID(path string, index int, text string) string`
Builds a deterministic MD5 hex digest from `path`, `index`, and the first 100 runes of `text`. The rune truncation keeps the prefix UTF-8 safe before hashing.

### `func FileExtensions() []string`
Returns the list of file extensions considered ingestible: `.md`, `.markdown`, and `.txt`.

### `func ShouldProcess(path string) bool`
Returns `true` when the lowercased extension of `path` is present in `FileExtensions()`.

### `func ListCollections(ctx context.Context, store VectorStore) ([]string, error)`
Delegates directly to `store.ListCollections(ctx)`.

### `func ListCollectionsSeq(ctx context.Context, store VectorStore) (iter.Seq[string], error)`
Fetches the collection list once and returns an iterator over the resulting slice. If `store.ListCollections` fails, the error is returned before the iterator is created.

### `func DeleteCollection(ctx context.Context, store VectorStore, name string) error`
Delegates directly to `store.DeleteCollection(ctx, name)`.

### `func CollectionStats(ctx context.Context, store VectorStore, name string) (*CollectionInfo, error)`
Delegates directly to `store.CollectionInfo(ctx, name)`.

### `func DefaultIngestConfig() IngestConfig`
Returns `IngestConfig{Collection: "hostuk-docs", BatchSize: 100, Chunk: DefaultChunkConfig()}`. `Directory`, `Recreate`, and `Verbose` remain at their zero values.

### `func Ingest(ctx context.Context, store VectorStore, embedder Embedder, cfg IngestConfig, progress IngestProgress) (*IngestStats, error)`
Recursively scans `cfg.Directory` or `.` for `.md`, `.markdown`, and `.txt` files; validates that the scan root exists and is a directory; optionally deletes and recreates the target collection; creates the collection when it does not already exist using `embedder.EmbedDimension()`; reads each file; skips empty content; chunks each document; embeds each chunk; builds `Point` payloads containing `text`, `source`, `section`, `category`, and `chunk_index`; and batch-upserts the resulting points. `stats.Errors` increments for unreadable files and embedding failures. The function returns an error when directory access fails, the scan root is not a directory, no matching files are found, collection operations fail, or any upsert batch fails.

### `func IngestFile(ctx context.Context, store VectorStore, embedder Embedder, collection string, filePath string, chunkCfg ChunkConfig) (int, error)`
Reads one file, returns `0, nil` for empty content, chunks the content with `chunkCfg`, embeds every chunk, creates `Point` payloads with the same metadata keys used by `Ingest`, and upserts them into `collection`. It does not create or verify the collection before writing.

### `func QueryWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) ([]QueryResult, error)`
Starts from `DefaultQueryConfig()`, overrides `Collection` and `Limit`, and forwards the call to `Query`.

### `func QueryContextWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) (string, error)`
Calls `QueryWith` and converts the resulting hits into the XML-like context format returned by `FormatResultsContext`.

### `func IngestDirWith(ctx context.Context, store VectorStore, embedder Embedder, directory, collectionName string, recreateCollection bool) error`
Starts from `DefaultIngestConfig()`, overrides `Directory`, `Collection`, and `Recreate`, then calls `Ingest` with no progress callback.

### `func IngestFileWith(ctx context.Context, store VectorStore, embedder Embedder, filePath, collectionName string) (int, error)`
Calls `IngestFile` with `DefaultChunkConfig()`.

### `func QueryDocs(ctx context.Context, question, collectionName string, topK int) ([]QueryResult, error)`
Creates a `QdrantClient` from `DefaultQdrantConfig()` and an `OllamaClient` from `DefaultOllamaConfig()`, closes the Qdrant client on return, and delegates to `QueryWith`.

### `func QueryDocsContext(ctx context.Context, question, collectionName string, topK int) (string, error)`
Calls `QueryDocs` and formats the results with `FormatResultsContext`.

### `func IngestDirectory(ctx context.Context, directory, collectionName string, recreateCollection bool) error`
Creates default Qdrant and Ollama clients, runs `QdrantClient.HealthCheck`, runs `OllamaClient.VerifyModel`, and then delegates to `IngestDirWith`.

### `func IngestSingleFile(ctx context.Context, filePath, collectionName string) (int, error)`
Creates default Qdrant and Ollama clients, runs the same health and model checks used by `IngestDirectory`, and then delegates to `IngestFileWith`.

### `func DefaultOllamaConfig() OllamaConfig`
Returns `OllamaConfig{Host: "localhost", Port: 11434, Model: "nomic-embed-text"}`.

### `func NewOllamaClient(cfg OllamaConfig) (*OllamaClient, error)`
Creates an Ollama HTTP client targeting `http://{Host}:{Port}` with a `30s` timeout and stores both the generated `api.Client` and the provided config in the returned `OllamaClient`.

### `func DefaultQdrantConfig() QdrantConfig`
Returns `QdrantConfig{Host: "localhost", Port: 6334, UseTLS: false}`. `APIKey` defaults to the empty string.

### `func NewQdrantClient(cfg QdrantConfig) (*QdrantClient, error)`
Creates a Qdrant client using the supplied host, port, API key, and TLS setting. Construction errors are wrapped with the resolved `host:port` address.

### `func DefaultQueryConfig() QueryConfig`
Returns `QueryConfig{Collection: "hostuk-docs", Limit: 5, Threshold: 0.5}`. `Category` and `Keywords` keep their zero values.

### `func Query(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) ([]QueryResult, error)`
Collects the iterator returned by `QuerySeq` into a slice.

### `func QuerySeq(ctx context.Context, store VectorStore, embedder Embedder, query string, cfg QueryConfig) (iter.Seq[QueryResult], error)`
Embeds the query text, builds a `category` filter when `cfg.Category` is non-empty, searches `cfg.Collection` with `cfg.Limit`, drops hits below `cfg.Threshold`, projects payload fields into `QueryResult`, normalizes `chunk_index` values coming back as `int64`, `float64`, or `int`, and optionally re-ranks the results with `KeywordFilter` when `cfg.Keywords` is `true`.

### `func FormatResultsText(results []QueryResult) string`
Returns `"No results found."` for empty input. Otherwise renders one plain-text block per result including score, source, optional section, category, and the chunk text.

### `func FormatResultsContext(results []QueryResult) string`
Returns an empty string for empty input. Otherwise emits a `<retrieved_context>` wrapper containing one `<document>` element per result. Source, section, category, and text content are escaped with `html.EscapeString`.

### `func FormatResultsJSON(results []QueryResult) string`
Returns `"[]"` for empty input. Otherwise serializes a JSON array containing `source`, `section`, `category`, `score`, and `text` for each result. Scores are rounded to four decimal places before encoding.

### `func KeywordFilter(results []QueryResult, keywords []string) []QueryResult`
Returns the original slice unchanged when either input slice is empty. Otherwise lowercases the keywords once, counts case-insensitive keyword matches in each result's text, multiplies the original score by `1 + 0.1*matchCount`, and returns a score-descending copy of the input results.

### `func KeywordFilterSeq(results []QueryResult, keywords []string) iter.Seq[QueryResult]`
Iterator wrapper over `KeywordFilter`.

### Methods

### `func (o *OllamaClient) EmbedDimension() uint64`
Returns a hard-coded vector size by model name: `768` for `nomic-embed-text`, `1024` for `mxbai-embed-large`, `384` for `all-minilm`, and `768` for any other model name.

### `func (o *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error)`
Calls Ollama's embed endpoint with the configured model and a single input string, rejects empty embedding responses, and converts the returned `float64` values to `float32`.

### `func (o *OllamaClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)`
Embeds each input sequentially by calling `Embed`. A failure on any item aborts the batch and returns a wrapped error that includes the failing index.

### `func (o *OllamaClient) VerifyModel(ctx context.Context) error`
Calls `Embed` with the literal text `"test"`. When that fails, it returns an error that includes `ollama pull {model}` guidance for the configured model name.

### `func (o *OllamaClient) Model() string`
Returns the configured model name unchanged.

### `func (q *QdrantClient) Close() error`
Closes the underlying Qdrant client connection.

### `func (q *QdrantClient) HealthCheck(ctx context.Context) error`
Runs the underlying client's health check and returns only the resulting error value.

### `func (q *QdrantClient) ListCollections(ctx context.Context) ([]string, error)`
Fetches collection names from Qdrant and returns a copied slice of those names.

### `func (q *QdrantClient) CollectionExists(ctx context.Context, name string) (bool, error)`
Delegates directly to the underlying Qdrant client's collection existence check.

### `func (q *QdrantClient) CreateCollection(ctx context.Context, name string, vectorSize uint64) error`
Creates a collection configured for cosine distance with the supplied vector size.

### `func (q *QdrantClient) DeleteCollection(ctx context.Context, name string) error`
Deletes the named collection through the underlying client.

### `func (q *QdrantClient) CollectionInfo(ctx context.Context, name string) (*CollectionInfo, error)`
Loads collection metadata from Qdrant, fills `Name` and `PointCount`, reads vector size from `VectorsConfig.Params` when present, and maps the Qdrant collection status enum to the exported status strings `green`, `yellow`, `red`, or `unknown`.

### `func (q *QdrantClient) UpsertPoints(ctx context.Context, collection string, points []Point) error`
Returns immediately when `points` is empty. Otherwise converts each `Point` into a `qdrant.PointStruct` with a string ID, dense vector, and payload map, then upserts the full batch into `collection`.

### `func (q *QdrantClient) Search(ctx context.Context, collection string, vector []float32, limit uint64, filter map[string]string) ([]SearchResult, error)`
Builds a `qdrant.QueryPoints` request with `WithPayload` enabled, adds one exact-match `Must` condition per filter entry when `filter` is non-empty, executes the query, converts Qdrant payload values into native Go values, and returns `SearchResult` values containing string IDs, scores, and converted payload maps.
