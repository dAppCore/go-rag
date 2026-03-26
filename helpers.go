package rag

import (
	"context"

	"dappco.re/go/core"
)

// QueryWith queries the vector store using the provided embedder and store.
// QueryWith(ctx, store, embedder, "how do goroutines work?", "project-docs", 5)
func QueryWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) ([]QueryResult, error) {
	cfg := DefaultQueryConfig()
	cfg.Collection = collectionName
	cfg.Limit = uint64(topK)

	return Query(ctx, store, embedder, question, cfg)
}

// QueryContextWith queries and returns context-formatted results using the
// provided embedder and store.
// QueryContextWith(ctx, store, embedder, "how do goroutines work?", "project-docs", 5)
func QueryContextWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) (string, error) {
	results, err := QueryWith(ctx, store, embedder, question, collectionName, topK)
	if err != nil {
		return "", err
	}
	return FormatResultsContext(results), nil
}

// IngestDirWith ingests all documents in a directory using the provided
// embedder and store.
// IngestDirWith(ctx, store, embedder, "./docs", "project-docs", true)
func IngestDirWith(ctx context.Context, store VectorStore, embedder Embedder, directory, collectionName string, recreateCollection bool) error {
	cfg := DefaultIngestConfig()
	cfg.Directory = directory
	cfg.Collection = collectionName
	cfg.Recreate = recreateCollection

	_, err := Ingest(ctx, store, embedder, cfg, nil)
	return err
}

// IngestFileWith ingests a single file using the provided embedder and store.
// IngestFileWith(ctx, store, embedder, "./docs/guide.md", "project-docs")
func IngestFileWith(ctx context.Context, store VectorStore, embedder Embedder, filePath, collectionName string) (int, error) {
	return IngestFile(ctx, store, embedder, collectionName, filePath, DefaultChunkConfig())
}

// QueryDocs queries the RAG database with default clients.
// QueryDocs(ctx, "how do goroutines work?", "project-docs", 5)
func QueryDocs(ctx context.Context, question, collectionName string, topK int) ([]QueryResult, error) {
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	if err != nil {
		return nil, err
	}
	defer func() { _ = qdrantClient.Close() }()

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	if err != nil {
		return nil, err
	}

	return QueryWith(ctx, qdrantClient, ollamaClient, question, collectionName, topK)
}

// QueryDocsContext queries the RAG database and returns context-formatted results.
// QueryDocsContext(ctx, "how do goroutines work?", "project-docs", 5)
func QueryDocsContext(ctx context.Context, question, collectionName string, topK int) (string, error) {
	results, err := QueryDocs(ctx, question, collectionName, topK)
	if err != nil {
		return "", err
	}
	return FormatResultsContext(results), nil
}

// IngestDirectory ingests all documents in a directory with default clients.
// IngestDirectory(ctx, "./docs", "project-docs", true)
func IngestDirectory(ctx context.Context, directory, collectionName string, recreateCollection bool) error {
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	if err != nil {
		return err
	}
	defer func() { _ = qdrantClient.Close() }()

	if err := qdrantClient.HealthCheck(ctx); err != nil {
		return core.E("rag.IngestDirectory", "qdrant health check failed", err)
	}

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	if err != nil {
		return err
	}

	if err := ollamaClient.VerifyModel(ctx); err != nil {
		return err
	}

	return IngestDirWith(ctx, qdrantClient, ollamaClient, directory, collectionName, recreateCollection)
}

// IngestSingleFile ingests a single file with default clients.
// IngestSingleFile(ctx, "./docs/guide.md", "project-docs")
func IngestSingleFile(ctx context.Context, filePath, collectionName string) (int, error) {
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	if err != nil {
		return 0, err
	}
	defer func() { _ = qdrantClient.Close() }()

	if err := qdrantClient.HealthCheck(ctx); err != nil {
		return 0, core.E("rag.IngestSingleFile", "qdrant health check failed", err)
	}

	ollamaClient, err := NewOllamaClient(DefaultOllamaConfig())
	if err != nil {
		return 0, err
	}

	if err := ollamaClient.VerifyModel(ctx); err != nil {
		return 0, err
	}

	return IngestFileWith(ctx, qdrantClient, ollamaClient, filePath, collectionName)
}
