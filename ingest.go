package rag

import (
	"context"
	"io"
	"io/fs"
	"slices"

	"dappco.re/go/core"
	"github.com/ledongthuc/pdf"
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

		content, readErr := readDocument(localFS, filePath)
		if readErr != nil {
			stats.Errors++
			continue
		}

		if core.Trim(content) == "" {
			continue
		}

		// Chunk the content
		category := Category(relPath)
		chunks := ChunkMarkdown(content, cfg.Chunk)

		for batch := range slices.Chunk(chunks, cfg.BatchSize) {
			texts := make([]string, len(batch))
			for i, chunk := range batch {
				texts[i] = chunk.Text
			}

			embeddings, err := embedChunkBatch(ctx, embedder, texts)
			if err != nil {
				for _, chunk := range batch {
					embedding, embedErr := embedder.Embed(ctx, chunk.Text)
					if embedErr != nil {
						stats.Errors++
						if cfg.Verbose {
							core.Print(nil, "  Error embedding %s chunk %d: %v", relPath, chunk.Index, embedErr)
						}
						continue
					}
					points = append(points, buildPoint(relPath, category, chunk, embedding))
					stats.Chunks++
				}
				continue
			}

			for i, chunk := range batch {
				points = append(points, buildPoint(relPath, category, chunk, embeddings[i]))
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

	content, readErr := readDocument(localFS, filePath)
	if readErr != nil {
		return 0, core.Wrap(readErr, "rag.IngestFile", "error reading file")
	}

	if core.Trim(content) == "" {
		return 0, nil
	}

	category := Category(filePath)
	chunks := ChunkMarkdown(content, chunkCfg)

	var points []Point
	for batch := range slices.Chunk(chunks, 100) {
		texts := make([]string, len(batch))
		for i, chunk := range batch {
			texts[i] = chunk.Text
		}

		embeddings, err := embedChunkBatch(ctx, embedder, texts)
		if err != nil {
			return 0, core.E("rag.IngestFile", "error embedding chunks", err)
		}

		for i, chunk := range batch {
			points = append(points, buildPoint(filePath, category, chunk, embeddings[i]))
		}
	}

	if err := store.UpsertPoints(ctx, collection, points); err != nil {
		return 0, core.E("rag.IngestFile", "error upserting points", err)
	}

	return len(points), nil
}

func embedChunkBatch(ctx context.Context, embedder Embedder, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	embeddings, err := embedder.EmbedBatch(ctx, texts)
	if err == nil && len(embeddings) == len(texts) {
		return embeddings, nil
	}
	if err == nil {
		err = core.E("rag.embedChunkBatch", core.Sprintf("unexpected embedding count: got %d, want %d", len(embeddings), len(texts)), nil)
	}

	// Fall back to sequential embedding so a batch-level failure does not
	// drop the entire file on the floor.
	embeddings = make([][]float32, len(texts))
	for i, text := range texts {
		vec, embedErr := embedder.Embed(ctx, text)
		if embedErr != nil {
			return nil, core.E("rag.embedChunkBatch", core.Sprintf("error embedding chunk %d", i), embedErr)
		}
		embeddings[i] = vec
	}
	return embeddings, nil
}

func buildPoint(source, category string, chunk Chunk, embedding []float32) Point {
	return Point{
		ID:     ChunkID(source, chunk.Index, chunk.Text),
		Vector: embedding,
		Payload: map[string]any{
			"text":        chunk.Text,
			"source":      source,
			"section":     chunk.Section,
			"category":    category,
			"chunk_index": chunk.Index,
		},
	}
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

// readDocument reads a file as text, with PDF extraction for .pdf extensions.
// Non-PDF files are read via the supplied Fs.
// PDFs that fail extraction fall back to reading as plain text when the error
// indicates a malformed/non-PDF file.
//
//	text, err := readDocument(fs, "./docs/guide.pdf")
func readDocument(fs *core.Fs, filePath string) (string, error) {
	if core.Lower(core.PathExt(filePath)) == ".pdf" {
		content, err := readPDFDocument(filePath)
		if err == nil && core.Trim(content) != "" {
			return content, nil
		}
		if err != nil && shouldFallbackToPlainText(err) {
			return readAsText(fs, filePath)
		}
		if err == nil {
			return content, nil
		}
		return "", err
	}
	return readAsText(fs, filePath)
}

func readAsText(fs *core.Fs, filePath string) (string, error) {
	result := fs.Read(filePath)
	if !result.OK {
		return "", resultError(result)
	}
	text, ok := result.Value.(string)
	if !ok {
		return "", core.E("rag.readDocument", "unexpected read result type", nil)
	}
	return text, nil
}

// readPDFDocument extracts plaintext from a PDF file.
func readPDFDocument(filePath string) (string, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	plainText, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(plainText)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// shouldFallbackToPlainText returns true when a PDF parse error indicates
// the file is actually plain text with a .pdf extension.
func shouldFallbackToPlainText(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return core.Contains(msg, "not a PDF file") || core.Contains(msg, "missing %%EOF")
}
