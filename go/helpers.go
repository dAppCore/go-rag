package rag

import (
	"context"
	"sync"

	"dappco.re/go"
)

const maxConcurrentEmbeddings = 8

type defaultQdrantClient interface {
	VectorStore
	Close() core.Result
	HealthCheck(context.Context) core.Result
}

type defaultOllamaClient interface {
	Embedder
	VerifyModel(context.Context) core.Result
}

var newDefaultQdrantClient = func() (defaultQdrantClient, error) {
	r := NewQdrantClient(DefaultQdrantConfig())
	if !r.OK {
		return nil, core.NewError(r.Error())
	}
	return r.Value.(*QdrantClient), nil
}

var newDefaultOllamaClient = func() (defaultOllamaClient, error) {
	r := NewOllamaClient(DefaultOllamaConfig())
	if !r.OK {
		return nil, core.NewError(r.Error())
	}
	return r.Value.(*OllamaClient), nil
}

// QueryWith queries the vector store using the provided embedder and store.
// QueryWith(ctx, store, embedder, "how do goroutines work?", "project-docs", 5)
func QueryWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) core.Result {
	cfg := DefaultQueryConfig()
	cfg.Collection = collectionName
	if topK < 0 {
		topK = 0
	}
	cfg.Limit = uint64(topK)

	return Query(ctx, store, embedder, question, cfg)
}

// QueryContextWith queries and returns context-formatted results using the
// provided embedder and store.
// QueryContextWith(ctx, store, embedder, "how do goroutines work?", "project-docs", 5)
func QueryContextWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) core.Result {
	resultsResult := QueryWith(ctx, store, embedder, question, collectionName, topK)
	if !resultsResult.OK {
		return resultsResult
	}
	results := resultsResult.Value.([]QueryResult)
	return core.Ok(FormatResultsContext(results))
}

// IngestDirWith ingests all documents in a directory using the provided
// embedder and store.
// IngestDirWith(ctx, store, embedder, "./docs", "project-docs", true)
func IngestDirWith(ctx context.Context, store VectorStore, embedder Embedder, directory, collectionName string, recreateCollection bool) core.Result {
	cfg := DefaultIngestConfig()
	cfg.Directory = directory
	cfg.Collection = collectionName
	cfg.Recreate = recreateCollection

	return Ingest(ctx, store, embedder, cfg, nil)
}

// IngestFileWith ingests a single file using the provided embedder and store.
// IngestFileWith(ctx, store, embedder, "./docs/guide.md", "project-docs")
func IngestFileWith(ctx context.Context, store VectorStore, embedder Embedder, filePath, collectionName string) core.Result {
	return IngestFile(ctx, store, embedder, collectionName, filePath, DefaultChunkConfig())
}

// QueryDocs queries the RAG database with default clients.
// QueryDocs(ctx, "how do goroutines work?", "project-docs", 5)
func QueryDocs(ctx context.Context, question, collectionName string, topK int) core.Result {
	qdrantClient, err := newDefaultQdrantClient()
	if err != nil {
		return core.Fail(err)
	}
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Warn("qdrant close failed", "err", r.Error())
		}
	}()

	ollamaClient, err := newDefaultOllamaClient()
	if err != nil {
		return core.Fail(err)
	}

	return QueryWith(ctx, qdrantClient, ollamaClient, question, collectionName, topK)
}

// QueryDocsContext queries the RAG database and returns context-formatted results.
// QueryDocsContext(ctx, "how do goroutines work?", "project-docs", 5)
func QueryDocsContext(ctx context.Context, question, collectionName string, topK int) core.Result {
	resultsResult := QueryDocs(ctx, question, collectionName, topK)
	if !resultsResult.OK {
		return resultsResult
	}
	results := resultsResult.Value.([]QueryResult)
	return core.Ok(FormatResultsContext(results))
}

// IngestDirectory ingests all documents in a directory with default clients.
// IngestDirectory(ctx, "./docs", "project-docs", true)
func IngestDirectory(ctx context.Context, directory, collectionName string, recreateCollection bool) core.Result {
	qdrantClient, err := newDefaultQdrantClient()
	if err != nil {
		return core.Fail(err)
	}
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Warn("qdrant close failed", "err", r.Error())
		}
	}()

	if r := qdrantClient.HealthCheck(ctx); !r.OK {
		return core.Fail(core.E("rag.IngestDirectory", "qdrant health check failed", core.NewError(r.Error())))
	}

	ollamaClient, err := newDefaultOllamaClient()
	if err != nil {
		return core.Fail(err)
	}

	if r := ollamaClient.VerifyModel(ctx); !r.OK {
		return r
	}

	return IngestDirWith(ctx, qdrantClient, ollamaClient, directory, collectionName, recreateCollection)
}

// IngestSingleFile ingests a single file with default clients.
// IngestSingleFile(ctx, "./docs/guide.md", "project-docs")
func IngestSingleFile(ctx context.Context, filePath, collectionName string) core.Result {
	qdrantClient, err := newDefaultQdrantClient()
	if err != nil {
		return core.Fail(err)
	}
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Warn("qdrant close failed", "err", r.Error())
		}
	}()

	if r := qdrantClient.HealthCheck(ctx); !r.OK {
		return core.Fail(core.E("rag.IngestSingleFile", "qdrant health check failed", core.NewError(r.Error())))
	}

	ollamaClient, err := newDefaultOllamaClient()
	if err != nil {
		return core.Fail(err)
	}

	if r := ollamaClient.VerifyModel(ctx); !r.OK {
		return r
	}

	return IngestFileWith(ctx, qdrantClient, ollamaClient, filePath, collectionName)
}

// textResult is implemented by any result type that can expose its text for
// prompt assembly. QueryResult and SearchResult both satisfy this interface.
//
//	var _ textResult = QueryResult{}
type textResult interface {
	GetText() string
}

// JoinResults concatenates result text into a single prompt-friendly string,
// skipping empty entries. Generic over anything that exposes GetText().
//
//	prompt := JoinResults(results)
func JoinResults[T textResult](results []T) string {
	if len(results) == 0 {
		return ""
	}
	parts := make([]string, 0, len(results))
	for _, result := range results {
		text := core.Trim(result.GetText())
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return core.Join("\n\n", parts...)
}

// embedBatchConcurrent runs embeddings in parallel while preserving input order.
// It returns the collected vectors and a per-input error slice for callers that
// need partial failure reporting.
func embedBatchConcurrent(ctx context.Context, texts []string, embed func(context.Context, string) core.Result) core.Result {
	if len(texts) == 0 {
		return core.Ok(embedChunkBatchResult{Embeddings: [][]float32{}, Results: []core.Result{}})
	}

	vectors := make([][]float32, len(texts))
	results := make([]core.Result, len(texts))
	workerCount := len(texts)
	if workerCount > maxConcurrentEmbeddings {
		workerCount = maxConcurrentEmbeddings
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(workerCount)
	for range workerCount {
		go func() {
			defer wg.Done()
			for i := range jobs {
				select {
				case <-ctx.Done():
					results[i] = core.Fail(ctx.Err())
					continue
				default:
				}

				vecResult := embed(ctx, texts[i])
				if !vecResult.OK {
					results[i] = vecResult
					continue
				}
				vec := vecResult.Value.([]float32)
				vectors[i] = vec
				results[i] = core.Ok(nil)
			}
		}()
	}

	for i := range texts {
		select {
		case <-ctx.Done():
			results[i] = core.Fail(ctx.Err())
		case jobs <- i:
		}
	}
	close(jobs)

	wg.Wait()
	return core.Ok(embedChunkBatchResult{Embeddings: vectors, Results: results})
}
