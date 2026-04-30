package rag

import (
	"context"
	"io"

	"dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	"dappco.re/go/rag"
)

var (
	queryCollection string
	limit           int
	threshold       float32
	category        string
	keywords        bool
	format          string
)

var queryCmd = &cli.Command{
	Use:   "query [question]",
	Short: "Query documents",
	Long:  "Embed a question and search a RAG collection for relevant context.",
	Args:  cli.ExactArgs(1),
	RunE: func(cmd *cli.Command, args []string) error {
		r := runQuery(cmd, args)
		if r.OK {
			return nil
		}
		if err, ok := r.Value.(error); ok {
			return err
		}
		return core.NewError(r.Error())
	},
}

// runQuery embeds a question, searches Qdrant, and writes formatted results.
func runQuery(cmd *cli.Command, args []string) core.Result {
	question := args[0]
	ctx := context.Background()

	// Connect to Qdrant
	qdrantResult := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if !qdrantResult.OK {
		return core.Fail(core.E("rag.cmd.query", "failed to connect to Qdrant", core.NewError(qdrantResult.Error())))
	}
	qdrantClient := qdrantResult.Value.(*rag.QdrantClient)
	defer func() {
		if r := qdrantClient.Close(); !r.OK {
			core.Print(nil, "Qdrant close failed: %s", r.Error())
		}
	}()

	// Connect to Ollama
	ollamaResult := rag.NewOllamaClient(rag.OllamaConfig{
		Host:  ollamaHost,
		Port:  ollamaPort,
		Model: model,
	})
	if !ollamaResult.OK {
		return core.Fail(core.E("rag.cmd.query", "failed to connect to Ollama", core.NewError(ollamaResult.Error())))
	}
	ollamaClient := ollamaResult.Value.(*rag.OllamaClient)

	// Configure query
	if limit < 0 {
		limit = 0
	}
	cfg := rag.QueryConfig{
		Collection: queryCollection,
		Limit:      uint64(limit),
		Threshold:  threshold,
		Category:   category,
		Keywords:   keywords,
	}

	// Run query
	queryResult := rag.Query(ctx, qdrantClient, ollamaClient, question, cfg)
	if !queryResult.OK {
		return queryResult
	}
	results := queryResult.Value.([]rag.QueryResult)

	// Format output
	out := cmd.OutOrStdout()
	switch format {
	case "json":
		_, _ = io.WriteString(out, rag.FormatResultsJSON(results)+"\n")
	case "context":
		_, _ = io.WriteString(out, rag.FormatResultsContext(results)+"\n")
	default:
		_, _ = io.WriteString(out, rag.FormatResultsText(results)+"\n")
	}

	return core.Ok(results)
}
