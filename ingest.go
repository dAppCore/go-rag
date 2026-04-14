package rag

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	coreio "dappco.re/go/core/io"
	"dappco.re/go/core/log"
)

// IngestConfig holds ingestion configuration.
type IngestConfig struct {
	Directory  string
	Collection string
	Recreate   bool
	Verbose    bool
	BatchSize  int
	Chunk      ChunkConfig
}

// DefaultIngestConfig returns default ingestion configuration.
func DefaultIngestConfig() IngestConfig {
	return IngestConfig{
		Collection: "hostuk-docs",
		BatchSize:  100,
		Chunk:      DefaultChunkConfig(),
	}
}

// IngestStats holds statistics from ingestion.
type IngestStats struct {
	Files  int
	Chunks int
	Errors int
}

// IngestProgress is called during ingestion to report progress.
type IngestProgress func(file string, chunks int, total int)

// Ingest processes a directory of documents and stores them in the vector store.
func Ingest(ctx context.Context, store VectorStore, embedder Embedder, cfg IngestConfig, progress IngestProgress) (*IngestStats, error) {
	stats := &IngestStats{}

	// Validate batch size to prevent infinite loop
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100 // Safe default
	}

	// Resolve directory
	absDir, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return nil, log.E("rag.Ingest", "error resolving directory", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return nil, log.E("rag.Ingest", "error accessing directory", err)
	}
	if !info.IsDir() {
		return nil, log.E("rag.Ingest", fmt.Sprintf("not a directory: %s", absDir), nil)
	}

	// Check/create collection
	exists, err := store.CollectionExists(ctx, cfg.Collection)
	if err != nil {
		return nil, log.E("rag.Ingest", "error checking collection", err)
	}

	if cfg.Recreate && exists {
		if err := store.DeleteCollection(ctx, cfg.Collection); err != nil {
			return nil, log.E("rag.Ingest", "error deleting collection", err)
		}
		exists = false
	}

	if !exists {
		vectorDim := embedder.EmbedDimension()
		if err := store.CreateCollection(ctx, cfg.Collection, vectorDim); err != nil {
			return nil, log.E("rag.Ingest", "error creating collection", err)
		}
	}

	// Find markdown files
	var files []string
	err = filepath.WalkDir(absDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && ShouldProcess(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, log.E("rag.Ingest", "error walking directory", err)
	}

	if len(files) == 0 {
		return nil, log.E("rag.Ingest", fmt.Sprintf("no markdown files found in %s", absDir), nil)
	}

	// Process files
	var points []Point
	for _, filePath := range files {
		relPath, err := filepath.Rel(absDir, filePath)
		if err != nil {
			stats.Errors++
			continue
		}

		content, err := coreio.Local.Read(filePath)
		if err != nil {
			stats.Errors++
			continue
		}

		if len(strings.TrimSpace(content)) == 0 {
			continue
		}

		// Chunk the content
		category := Category(relPath)
		chunks := ChunkMarkdown(content, cfg.Chunk)
		chunkTexts := make([]string, len(chunks))
		for i, chunk := range chunks {
			chunkTexts[i] = chunk.Text
		}

		embeddings, err := embedder.EmbedBatch(ctx, chunkTexts)
		if err != nil {
			stats.Errors++
			if cfg.Verbose {
				fmt.Printf("  Error embedding %s: %v\n", relPath, err)
			}
		} else if len(embeddings) != len(chunks) {
			stats.Errors++
			if cfg.Verbose {
				fmt.Printf("  Error embedding %s: embedder returned %d vectors for %d chunks\n", relPath, len(embeddings), len(chunks))
			}
		} else {
			for i, chunk := range chunks {
				points = append(points, Point{
					ID:     ChunkID(relPath, chunk.Index, chunk.Text),
					Vector: embeddings[i],
					Payload: map[string]any{
						"text":        chunk.Text,
						"source":      relPath,
						"section":     chunk.Section,
						"category":    category,
						"chunk_index": chunk.Index,
					},
				})
				stats.Chunks++
			}
		}

		stats.Files++
		if progress != nil {
			progress(relPath, stats.Chunks, len(files))
		}
	}

	// Batch upsert to vector store
	if len(points) > 0 {
		for batch := range slices.Chunk(points, cfg.BatchSize) {
			if err := store.UpsertPoints(ctx, cfg.Collection, batch); err != nil {
				return stats, log.E("rag.Ingest", "error upserting batch", err)
			}
		}
	}

	return stats, nil
}

// IngestFile processes a single file and stores it in the vector store.
func IngestFile(ctx context.Context, store VectorStore, embedder Embedder, collection string, filePath string, chunkCfg ChunkConfig) (int, error) {
	content, err := coreio.Local.Read(filePath)
	if err != nil {
		return 0, log.E("rag.IngestFile", "error reading file", err)
	}

	if len(strings.TrimSpace(content)) == 0 {
		return 0, nil
	}

	category := Category(filePath)
	chunks := ChunkMarkdown(content, chunkCfg)
	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		chunkTexts[i] = chunk.Text
	}

	embeddings, err := embedder.EmbedBatch(ctx, chunkTexts)
	if err != nil {
		return 0, log.E("rag.IngestFile", "error embedding chunks", err)
	}
	if len(embeddings) != len(chunks) {
		return 0, log.E("rag.IngestFile", fmt.Sprintf("embedder returned %d vectors for %d chunks", len(embeddings), len(chunks)), nil)
	}

	var points []Point
	for i, chunk := range chunks {
		points = append(points, Point{
			ID:     ChunkID(filePath, chunk.Index, chunk.Text),
			Vector: embeddings[i],
			Payload: map[string]any{
				"text":        chunk.Text,
				"source":      filePath,
				"section":     chunk.Section,
				"category":    category,
				"chunk_index": chunk.Index,
			},
		})
	}

	if err := store.UpsertPoints(ctx, collection, points); err != nil {
		return 0, log.E("rag.IngestFile", "error upserting points", err)
	}

	return len(points), nil
}
