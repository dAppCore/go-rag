package rag

import (
	"context"
	"strings"

	"dappco.re/go/core/log"
)

// QueryWith queries the vector store using the provided embedder and store.
func QueryWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) ([]QueryResult, error) {
	cfg := DefaultQueryConfig()
	cfg.Collection = collectionName
	cfg.Limit = uint64(topK)

	return Query(ctx, store, embedder, question, cfg)
}

// QueryContextWith queries and returns context-formatted results using the
// provided embedder and store.
func QueryContextWith(ctx context.Context, store VectorStore, embedder Embedder, question, collectionName string, topK int) (string, error) {
	results, err := QueryWith(ctx, store, embedder, question, collectionName, topK)
	if err != nil {
		return "", err
	}
	return FormatResultsContext(results), nil
}

// IngestDirWith ingests all documents in a directory using the provided
// embedder and store.
func IngestDirWith(ctx context.Context, store VectorStore, embedder Embedder, directory, collectionName string, recreateCollection bool) error {
	cfg := DefaultIngestConfig()
	cfg.Directory = directory
	cfg.Collection = collectionName
	cfg.Recreate = recreateCollection

	_, err := Ingest(ctx, store, embedder, cfg, nil)
	return err
}

// IngestFileWith ingests a single file using the provided embedder and store.
func IngestFileWith(ctx context.Context, store VectorStore, embedder Embedder, filePath, collectionName string) (int, error) {
	return IngestFile(ctx, store, embedder, collectionName, filePath, DefaultChunkConfig())
}

// QueryDocs queries the RAG database with default clients.
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
func QueryDocsContext(ctx context.Context, question, collectionName string, topK int) (string, error) {
	results, err := QueryDocs(ctx, question, collectionName, topK)
	if err != nil {
		return "", err
	}
	return FormatResultsContext(results), nil
}

// IngestDirectory ingests all documents in a directory with default clients.
func IngestDirectory(ctx context.Context, directory, collectionName string, recreateCollection bool) error {
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	if err != nil {
		return err
	}
	defer func() { _ = qdrantClient.Close() }()

	if err := qdrantClient.HealthCheck(ctx); err != nil {
		return log.E("rag.IngestDirectory", "qdrant health check failed", err)
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
func IngestSingleFile(ctx context.Context, filePath, collectionName string) (int, error) {
	qdrantClient, err := NewQdrantClient(DefaultQdrantConfig())
	if err != nil {
		return 0, err
	}
	defer func() { _ = qdrantClient.Close() }()

	if err := qdrantClient.HealthCheck(ctx); err != nil {
		return 0, log.E("rag.IngestSingleFile", "qdrant health check failed", err)
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

// JoinResults concatenates result text into a single prompt-friendly string.
func JoinResults(results []QueryResult) string {
	if len(results) == 0 {
		return ""
	}

	parts := make([]string, 0, len(results))
	for _, result := range results {
		text := strings.TrimSpace(result.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}

	return strings.Join(parts, "\n\n")
}
