package rag

import (
	"context"
	"io"
	"io/fs"
	"slices"

	"dappco.re/go"
	"github.com/ledongthuc/pdf"
)

// IngestConfig holds ingestion configuration.
// cfg := IngestConfig{Directory: "./docs", Collection: "project-docs", BatchSize: 100}
type IngestConfig struct {
	// Directory is the root directory scanned for ingestible files.
	Directory string
	// Collection is the vector-store collection that receives ingested points.
	Collection string
	// Recreate deletes and recreates Collection before ingestion when true.
	Recreate bool
	// Verbose enables per-file progress output.
	Verbose bool
	// BatchSize controls embedding and upsert batch sizes.
	BatchSize int
	// Chunk configures Markdown chunking before embedding.
	Chunk ChunkConfig
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
	// Files counts non-empty files processed by ingestion.
	Files int
	// Chunks counts chunks successfully embedded and queued for upsert.
	Chunks int
	// Errors counts read and embedding failures that ingestion skipped.
	Errors int
}

// IngestProgress is called during ingestion to report progress.
// progress := IngestProgress(func(file string, chunks int, total int) {})
type IngestProgress func(file string, chunks int, total int)

type embedChunkBatchResult struct {
	Embeddings [][]float32
	Results    []core.Result
}

// Ingest processes a directory of documents and stores them in the vector store.
// Ingest(ctx, store, embedder, cfg, progress)
func Ingest(ctx context.Context, store VectorStore, embedder Embedder, cfg IngestConfig, progress IngestProgress) core.Result {
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
		return core.Fail(core.Wrap(core.NewError(infoResult.Error()), "rag.Ingest", "error accessing directory"))
	}
	info, ok := infoResult.Value.(fs.FileInfo)
	if !ok {
		return core.Fail(core.E("rag.Ingest", core.Sprintf("unexpected stat result type: %T", infoResult.Value), nil))
	}
	if !info.IsDir() {
		return core.Fail(core.E("rag.Ingest", core.Sprintf("not a directory: %s", scanRoot), nil))
	}

	// Check/create collection
	existsResult := store.CollectionExists(ctx, cfg.Collection)
	if !existsResult.OK {
		return core.Fail(core.E("rag.Ingest", "error checking collection", core.NewError(existsResult.Error())))
	}
	exists := existsResult.Value.(bool)

	if cfg.Recreate && exists {
		if r := store.DeleteCollection(ctx, cfg.Collection); !r.OK {
			return core.Fail(core.E("rag.Ingest", "error deleting collection", core.NewError(r.Error())))
		}
		exists = false
	}

	if !exists {
		vectorDim := embedder.EmbedDimension()
		if r := store.CreateCollection(ctx, cfg.Collection, vectorDim); !r.OK {
			return core.Fail(core.E("rag.Ingest", "error creating collection", core.NewError(r.Error())))
		}
	}

	// Find ingestible files.
	var files []string
	if r := collectMarkdownFiles(localFS, scanRoot, "", &files); !r.OK {
		return core.Fail(core.E("rag.Ingest", "error walking directory", core.NewError(r.Error())))
	}
	slices.Sort(files)

	if len(files) == 0 {
		return core.Fail(core.E("rag.Ingest", core.Sprintf("no matching files found in %s", scanRoot), nil))
	}

	// Process files
	var points []Point
	for _, relPath := range files {
		filePath := relPath
		if scanRoot != "." {
			filePath = core.JoinPath(scanRoot, relPath)
		}

		contentResult := readDocument(localFS, filePath)
		if !contentResult.OK {
			stats.Errors++
			if cfg.Verbose {
				core.Print(nil, "  Error reading %s: %s", relPath, contentResult.Error())
			}
			continue
		}
		content := contentResult.Value.(string)

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

			batchResult := embedChunkBatch(ctx, embedder, texts)
			batchPayload := batchResult.Value.(embedChunkBatchResult)
			for i, chunk := range batch {
				if !batchPayload.Results[i].OK {
					stats.Errors++
					if cfg.Verbose {
						core.Print(nil, "  Error embedding %s chunk %d: %s", relPath, chunk.Index, batchPayload.Results[i].Error())
					}
					continue
				}
				points = append(points, buildPoint(relPath, category, chunk, batchPayload.Embeddings[i]))
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
			if r := store.UpsertPoints(ctx, cfg.Collection, batch); !r.OK {
				return core.Fail(core.E("rag.Ingest", "error upserting batch", core.NewError(r.Error())))
			}
		}
	}

	return core.Ok(stats)
}

// IngestFile processes a single file and stores it in the vector store.
// IngestFile(ctx, store, embedder, "project-docs", "./docs/guide.md", DefaultChunkConfig())
func IngestFile(ctx context.Context, store VectorStore, embedder Embedder, collection string, filePath string, chunkCfg ChunkConfig) core.Result {
	localFS := (&core.Fs{}).NewUnrestricted()

	contentResult := readDocument(localFS, filePath)
	if !contentResult.OK {
		return core.Fail(core.Wrap(core.NewError(contentResult.Error()), "rag.IngestFile", "error reading file"))
	}
	content := contentResult.Value.(string)

	if core.Trim(content) == "" {
		return core.Ok(0)
	}

	category := Category(filePath)
	chunks := ChunkMarkdown(content, chunkCfg)

	var points []Point
	for batch := range slices.Chunk(chunks, 100) {
		texts := make([]string, len(batch))
		for i, chunk := range batch {
			texts[i] = chunk.Text
		}

		batchResult := embedChunkBatch(ctx, embedder, texts)
		batchPayload := batchResult.Value.(embedChunkBatchResult)
		for i, r := range batchPayload.Results {
			if !r.OK {
				return core.Fail(core.E("rag.IngestFile", core.Sprintf("error embedding chunk %d", i), core.NewError(r.Error())))
			}
		}

		for i, chunk := range batch {
			points = append(points, buildPoint(filePath, category, chunk, batchPayload.Embeddings[i]))
		}
	}

	if r := store.UpsertPoints(ctx, collection, points); !r.OK {
		return core.Fail(core.E("rag.IngestFile", "error upserting points", core.NewError(r.Error())))
	}

	return core.Ok(len(points))
}

// embedChunkBatch prefers the batch embedding API and falls back to
// per-item embedding so partial failures can still be reported.
func embedChunkBatch(ctx context.Context, embedder Embedder, texts []string) core.Result {
	if len(texts) == 0 {
		return core.Ok(embedChunkBatchResult{Embeddings: [][]float32{}, Results: []core.Result{}})
	}

	embeddingsResult := embedder.EmbedBatch(ctx, texts)
	if embeddingsResult.OK {
		embeddings := embeddingsResult.Value.([][]float32)
		if len(embeddings) == len(texts) {
			results := make([]core.Result, len(texts))
			for i := range results {
				results[i] = core.Ok(nil)
			}
			return core.Ok(embedChunkBatchResult{Embeddings: embeddings, Results: results})
		}
	}

	batchResult := embedBatchConcurrent(ctx, texts, embedder.Embed)
	batchPayload := batchResult.Value.(embedChunkBatchResult)
	for i, r := range batchPayload.Results {
		if !r.OK {
			batchPayload.Results[i] = core.Fail(core.E("rag.embedChunkBatch", core.Sprintf("error embedding chunk %d", i), core.NewError(r.Error())))
		}
	}
	return core.Ok(batchPayload)
}

// buildPoint converts a chunk and embedding into vector-store payload form.
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

// collectMarkdownFiles appends ingestible file paths below currentPath.
func collectMarkdownFiles(localFS *core.Fs, currentPath string, currentRel string, files *[]string) core.Result {
	listResult := localFS.List(currentPath)
	if !listResult.OK {
		return listResult
	}

	entries, ok := listResult.Value.([]fs.DirEntry)
	if !ok {
		return core.Fail(core.E("rag.collectMarkdownFiles", core.Sprintf("unexpected list result type: %T", listResult.Value), nil))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		switch {
		case a.Name() < b.Name():
			return -1
		case a.Name() > b.Name():
			return 1
		}
		return 0
	})
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
			if r := collectMarkdownFiles(localFS, childPath, childRel, files); !r.OK {
				return r
			}
			continue
		}

		if ShouldProcess(childRel) {
			*files = append(*files, childRel)
		}
	}

	return core.Ok(nil)
}

// readDocument reads a file as text, with PDF extraction for .pdf extensions.
// Non-PDF files are read via the supplied Fs.
// PDFs that fail extraction fall back to reading as plain text when the error
// indicates a malformed/non-PDF file.
//
//	text, err := readDocument(fs, "./docs/guide.pdf")
func readDocument(fs *core.Fs, filePath string) core.Result {
	if core.Lower(core.PathExt(filePath)) == ".pdf" {
		contentResult := readPDFDocument(filePath)
		if contentResult.OK {
			content := contentResult.Value.(string)
			if core.Trim(content) != "" {
				return core.Ok(content)
			}
			return core.Ok(content)
		}
		if shouldFallbackToPlainText(core.NewError(contentResult.Error())) {
			return readAsText(fs, filePath)
		}
		return contentResult
	}
	return readAsText(fs, filePath)
}

// readAsText reads a file through core.Fs and validates the string payload.
func readAsText(fs *core.Fs, filePath string) core.Result {
	result := fs.Read(filePath)
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Fail(err)
		}
		return core.Fail(core.E("rag.readDocument", result.Error(), nil))
	}
	text, ok := result.Value.(string)
	if !ok {
		return core.Fail(core.E("rag.readDocument", "unexpected read result type", nil))
	}
	return core.Ok(text)
}

// readPDFDocument extracts plaintext from a PDF file.
func readPDFDocument(filePath string) core.Result {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return core.Fail(err)
	}
	defer func() { _ = file.Close() }()

	plainText, err := reader.GetPlainText()
	if err != nil {
		return core.Fail(err)
	}

	data, err := io.ReadAll(plainText)
	if err != nil {
		return core.Fail(err)
	}

	return core.Ok(string(data))
}

// shouldFallbackToPlainText returns true when a PDF parse error indicates
// the file is actually plain text with a .pdf extension.
func shouldFallbackToPlainText(err error) bool {
	if err == nil {
		return false
	}
	msg := core.Lower(err.Error())
	if !core.Contains(msg, "pdf") {
		return false
	}

	return core.Contains(msg, "not a pdf file") ||
		core.Contains(msg, "missing %%eof") ||
		core.Contains(msg, "unexpected eof") ||
		core.Contains(msg, "invalid pdf") ||
		core.Contains(msg, "malformed pdf") ||
		core.Contains(msg, "no pdf header") ||
		core.Contains(msg, "header")
}
