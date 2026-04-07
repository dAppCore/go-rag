package rag

import (
	"context"
	"fmt"

	"dappco.re/go/core/cli/pkg/cli"
	"dappco.re/go/core/i18n"
	"dappco.re/go/core/log"
	"dappco.re/go/core/rag"
)

var (
	queryCollection string
	limit           int
	threshold       float32
	category        string
	format          string
)

var queryCmd = &cli.Command{
	Use:   "query [question]",
	Short: i18n.T("cmd.rag.query.short"),
	Long:  i18n.T("cmd.rag.query.long"),
	Args:  cli.ExactArgs(1),
	RunE:  runQuery,
}

func runQuery(cmd *cli.Command, args []string) error {
	question := args[0]
	ctx := context.Background()

	// Connect to Qdrant
	qdrantClient, err := rag.NewQdrantClient(rag.QdrantConfig{
		Host:   qdrantHost,
		Port:   qdrantPort,
		UseTLS: false,
	})
	if err != nil {
		return log.E("rag.cmd.query", "failed to connect to Qdrant", err)
	}
	defer func() { _ = qdrantClient.Close() }()

	// Connect to Ollama
	ollamaClient, err := rag.NewOllamaClient(rag.OllamaConfig{
		Host:  ollamaHost,
		Port:  ollamaPort,
		Model: model,
	})
	if err != nil {
		return log.E("rag.cmd.query", "failed to connect to Ollama", err)
	}

	// Configure query
	if limit < 0 {
		limit = 0
	}
	cfg := rag.QueryConfig{
		Collection: queryCollection,
		Limit:      uint64(limit),
		Threshold:  threshold,
		Category:   category,
	}

	// Run query
	results, err := rag.Query(ctx, qdrantClient, ollamaClient, question, cfg)
	if err != nil {
		return err
	}

	// Format output
	switch format {
	case "json":
		fmt.Println(rag.FormatResultsJSON(results))
	case "context":
		fmt.Println(rag.FormatResultsContext(results))
	default:
		fmt.Println(rag.FormatResultsText(results))
	}

	return nil
}
