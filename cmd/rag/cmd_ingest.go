package rag

import (
	"context"
	"io"

	"dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/i18n"
	"dappco.re/go/rag"
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
	RunE: func(cmd *cli.Command, args []string) error {
		r := runIngest(cmd, args)
		if r.OK {
			return nil
		}
		if err, ok := r.Value.(error); ok {
			return err
		}
		return core.NewError(r.Error())
	},
}

// runIngest validates local flags, connects clients, and ingests documents.
func runIngest(cmd *cli.Command, args []string) core.Result {
	directory := "."
	if len(args) > 0 {
		directory = args[0]
	}

	ctx := context.Background()
	out := cmd.OutOrStdout()

	// Validate local config before opening network clients.
	if chunkSize <= 0 {
		return core.Fail(core.E("rag.cmd.ingest", "chunk-size must be > 0", nil))
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		return core.Fail(core.E("rag.cmd.ingest", "chunk-overlap must be >= 0 and < chunk-size", nil))
	}

	// Connect to Qdrant
	core.Print(out, "Connecting to Qdrant at %s:%d...", qdrantHost, qdrantPort)
	qdrantResult := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if !qdrantResult.OK {
		return core.Fail(core.E("rag.cmd.ingest", "failed to connect to Qdrant", core.NewError(qdrantResult.Error())))
	}
	qdrantClient := qdrantResult.Value.(*rag.QdrantClient)
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Print(out, "Qdrant close failed: %s", r.Error())
		}
	}()

	if r := qdrantClient.HealthCheck(ctx); !r.OK {
		return core.Fail(core.E("rag.cmd.ingest", "qdrant health check failed", core.NewError(r.Error())))
	}

	// Connect to Ollama
	core.Print(out, "Using embedding model: %s (via %s:%d)", model, ollamaHost, ollamaPort)
	ollamaResult := rag.NewOllamaClient(rag.OllamaConfig{
		Host:  ollamaHost,
		Port:  ollamaPort,
		Model: model,
	})
	if !ollamaResult.OK {
		return core.Fail(core.E("rag.cmd.ingest", "failed to connect to Ollama", core.NewError(ollamaResult.Error())))
	}
	ollamaClient := ollamaResult.Value.(*rag.OllamaClient)

	if r := ollamaClient.VerifyModel(ctx); !r.OK {
		return r
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
			core.Print(out, "  Processed: %s (%d chunks total)", file, chunks)
		} else {
			_, _ = io.WriteString(out, core.Sprintf("\r  %s (%d chunks)    ", cli.DimStyle.Render(file), chunks))
		}
	}

	// Run ingestion
	_, _ = io.WriteString(out, core.Sprintf("\nIngesting from: %s\n", directory))
	if recreate {
		core.Print(out, "  (recreating collection: %s)", collection)
	}

	ingestResult := rag.Ingest(ctx, qdrantClient, ollamaClient, cfg, progress)
	if !ingestResult.OK {
		return ingestResult
	}
	stats := ingestResult.Value.(*rag.IngestStats)

	// Summary
	_, _ = io.WriteString(out, core.Sprintf("\n\n%s\n", cli.TitleStyle.Render("Ingestion complete!")))
	core.Print(out, "  Files processed: %d", stats.Files)
	core.Print(out, "  Chunks created:  %d", stats.Chunks)
	if stats.Errors > 0 {
		core.Print(out, "  Errors:          %s", cli.ErrorStyle.Render(core.Sprintf("%d", stats.Errors)))
	}
	core.Print(out, "  Collection:      %s", collection)

	return core.Ok(stats)
}
