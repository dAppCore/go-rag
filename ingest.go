package rag

import (
	"context"
	"io/fs"
	"slices"

	"dappco.re/go/core"
)

// IngestConfig holds ingestion configuration.
// cfg := IngestConfig{Directory: "./docs", Collection: "project-docs", BatchSize: 100}
type IngestConfig struct {
	Directory  string
	Collection string
	Recreate   bool
	Verbose    bool
	BatchSize  int
	Chunk      ChunkConfig
}

// DefaultIngestConfig returns default ingestion configuration.
// cfg := DefaultIngestConfig()
func DefaultIngestConfig() IngestConfig {
	return IngestConfig{
		Collection: "hostuk-docs",
		BatchSize:  100,
		Chunk:      DefaultChunkConfig(),
	}
}

// IngestStats holds statistics from ingestion.
// stats := IngestStats{Files: 12, Chunks: 84, Errors: 0}
type IngestStats struct {
	Files  int
	Chunks int
	Errors int
}

// IngestProgress is called during ingestion to report progress.
// progress := IngestProgress(func(file string, chunks int, total int) {})
type IngestProgress func(file string, chunks int, total int)

// Ingest processes a directory of documents and stores them in the vector store.
// Ingest(ctx, store, embedder, cfg, progress)
func Ingest(ctx context.Context, store VectorStore, embedder Embedder, cfg IngestConfig, progress IngestProgress) (*IngestStats, error) {
	stats := &IngestStats{}
	localFS := (&core.Fs{}).NewUnrestricted()

	// Validate batch size to prevent infinite loop
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100 // Safe default
	}

	scanRoot := cfg.Directory
	if scanRoot == "" {
		scanRoot = "."
	}

	infoResult := localFS.Stat(scanRoot)
	if !infoResult.OK {
		return nil, core.Wrap(resultError(infoResult), "rag.Ingest", "error accessing directory")
	}
	info := infoResult.Value.(fs.FileInfo)
	if !info.IsDir() {
		return nil, core.E("rag.Ingest", core.Sprintf("not a directory: %s", scanRoot), nil)
	}

	// Check/create collection
	exists, err := store.CollectionExists(ctx, cfg.Collection)
	if err != nil {
		return nil, core.E("rag.Ingest", "error checking collection", err)
	}

	if cfg.Recreate && exists {
		if err := store.DeleteCollection(ctx, cfg.Collection); err != nil {
			return nil, core.E("rag.Ingest", "error deleting collection", err)
		}
		exists = false
	}

	if !exists {
		vectorDim := embedder.EmbedDimension()
		if err := store.CreateCollection(ctx, cfg.Collection, vectorDim); err != nil {
			return nil, core.E("rag.Ingest", "error creating collection", err)
		}
	}

	// Find markdown files
	var files []string
	if err := collectMarkdownFiles(localFS, scanRoot, "", &files); err != nil {
		return nil, core.E("rag.Ingest", "error walking directory", err)
	}

	if len(files) == 0 {
		return nil, core.E("rag.Ingest", core.Sprintf("no markdown files found in %s", scanRoot), nil)
	}

	// Process files
	var points []Point
	for _, relPath := range files {
		filePath := relPath
		if scanRoot != "." {
			filePath = core.JoinPath(scanRoot, relPath)
		}

		contentResult := localFS.Read(filePath)
		if !contentResult.OK {
			stats.Errors++
			continue
		}
		content := contentResult.Value.(string)

		if core.Trim(content) == "" {
			continue
		}

		// Chunk the content
		category := Category(relPath)
		chunks := ChunkMarkdown(content, cfg.Chunk)

		for _, chunk := range chunks {
			// Generate embedding
			embedding, err := embedder.Embed(ctx, chunk.Text)
			if err != nil {
				stats.Errors++
				if cfg.Verbose {
					core.Print(nil, "  Error embedding %s chunk %d: %v", relPath, chunk.Index, err)
				}
				continue
			}

			// Create point
			points = append(points, Point{
				ID:     ChunkID(relPath, chunk.Index, chunk.Text),
				Vector: embedding,
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

		stats.Files++
		if progress != nil {
			progress(relPath, stats.Chunks, len(files))
		}
	}

	// Batch upsert to vector store
	if len(points) > 0 {
		for batch := range slices.Chunk(points, cfg.BatchSize) {
			if err := store.UpsertPoints(ctx, cfg.Collection, batch); err != nil {
				return stats, core.E("rag.Ingest", "error upserting batch", err)
			}
		}
	}

	return stats, nil
}

// IngestFile processes a single file and stores it in the vector store.
// IngestFile(ctx, store, embedder, "project-docs", "./docs/guide.md", DefaultChunkConfig())
func IngestFile(ctx context.Context, store VectorStore, embedder Embedder, collection string, filePath string, chunkCfg ChunkConfig) (int, error) {
	localFS := (&core.Fs{}).NewUnrestricted()

	contentResult := localFS.Read(filePath)
	if !contentResult.OK {
		return 0, core.Wrap(resultError(contentResult), "rag.IngestFile", "error reading file")
	}
	content := contentResult.Value.(string)

	if core.Trim(content) == "" {
		return 0, nil
	}

	category := Category(filePath)
	chunks := ChunkMarkdown(content, chunkCfg)

	var points []Point
	for _, chunk := range chunks {
		embedding, err := embedder.Embed(ctx, chunk.Text)
		if err != nil {
			return 0, core.E("rag.IngestFile", core.Sprintf("error embedding chunk %d", chunk.Index), err)
		}

		points = append(points, Point{
			ID:     ChunkID(filePath, chunk.Index, chunk.Text),
			Vector: embedding,
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
		return 0, core.E("rag.IngestFile", "error upserting points", err)
	}

	return len(points), nil
}

func collectMarkdownFiles(localFS *core.Fs, currentPath string, currentRel string, files *[]string) error {
	listResult := localFS.List(currentPath)
	if !listResult.OK {
		return resultError(listResult)
	}

	entries := listResult.Value.([]fs.DirEntry)
	for _, entry := range entries {
		childPath := entry.Name()
		if currentPath != "." && currentPath != "" {
			childPath = core.JoinPath(currentPath, entry.Name())
		}

		childRel := entry.Name()
		if currentRel != "" {
			childRel = core.JoinPath(currentRel, entry.Name())
		}

		if entry.IsDir() {
			if err := collectMarkdownFiles(localFS, childPath, childRel, files); err != nil {
				return err
			}
			continue
		}

		if ShouldProcess(childRel) {
			*files = append(*files, childRel)
		}
	}

	return nil
}

func resultError(result core.Result) error {
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("rag.result", "core operation failed", nil)
}
