package rag

import (
	"context"
	"fmt"

	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-i18n"
	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-rag"
)

var (
	collection   string
	recreate     bool
	chunkSize    int
	chunkOverlap int
)

var ingestCmd = &cli.Command{
	Use:   "ingest [directory]",
	Short: i18n.T("cmd.rag.ingest.short"),
	Long:  i18n.T("cmd.rag.ingest.long"),
	Args:  cli.MaximumNArgs(1),
	RunE:  runIngest,
}

func runIngest(cmd *cli.Command, args []string) error {
	directory := "."
	if len(args) > 0 {
		directory = args[0]
	}

	ctx := context.Background()

	// Connect to Qdrant
	fmt.Printf("Connecting to Qdrant at %s:%d...\n", qdrantHost, qdrantPort)
	qdrantClient, err := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if err != nil {
		return log.E("rag.cmd.ingest", "failed to connect to Qdrant", err)
	}
	defer func() { _ = qdrantClient.Close() }()

	if err := qdrantClient.HealthCheck(ctx); err != nil {
		return log.E("rag.cmd.ingest", "qdrant health check failed", err)
	}

	// Connect to Ollama
	fmt.Printf("Using embedding model: %s (via %s:%d)\n", model, ollamaHost, ollamaPort)
	ollamaClient, err := rag.NewOllamaClient(rag.OllamaConfig{
		Host:  ollamaHost,
		Port:  ollamaPort,
		Model: model,
	})
	if err != nil {
		return log.E("rag.cmd.ingest", "failed to connect to Ollama", err)
	}

	if err := ollamaClient.VerifyModel(ctx); err != nil {
		return err
	}

	// Configure ingestion
	if chunkSize <= 0 {
		return log.E("rag.cmd.ingest", "chunk-size must be > 0", nil)
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		return log.E("rag.cmd.ingest", "chunk-overlap must be >= 0 and < chunk-size", nil)
	}

	cfg := rag.IngestConfig{
		Directory:  directory,
		Collection: collection,
		Recreate:   recreate,
		Verbose:    verbose,
		BatchSize:  100,
		Chunk: rag.ChunkConfig{
			Size:    chunkSize,
			Overlap: chunkOverlap,
		},
	}

	// Progress callback
	progress := func(file string, chunks int, total int) {
		if verbose {
			fmt.Printf("  Processed: %s (%d chunks total)\n", file, chunks)
		} else {
			fmt.Printf("\r  %s (%d chunks)    ", cli.DimStyle.Render(file), chunks)
		}
	}

	// Run ingestion
	fmt.Printf("\nIngesting from: %s\n", directory)
	if recreate {
		fmt.Printf("  (recreating collection: %s)\n", collection)
	}

	stats, err := rag.Ingest(ctx, qdrantClient, ollamaClient, cfg, progress)
	if err != nil {
		return err
	}

	// Summary
	fmt.Printf("\n\n%s\n", cli.TitleStyle.Render("Ingestion complete!"))
	fmt.Printf("  Files processed: %d\n", stats.Files)
	fmt.Printf("  Chunks created:  %d\n", stats.Chunks)
	if stats.Errors > 0 {
		fmt.Printf("  Errors:          %s\n", cli.ErrorStyle.Render(fmt.Sprintf("%d", stats.Errors)))
	}
	fmt.Printf("  Collection:      %s\n", collection)

	return nil
}
