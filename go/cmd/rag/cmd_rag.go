package rag

import (
	"strconv"
	"sync"

	"dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

// Shared flags
var (
	qdrantHost string
	qdrantPort int
	ollamaHost string
	ollamaPort int
	model      string
	verbose    bool
)

var initFlagsOnce sync.Once

var ragCmd = &cli.Command{
	Use:   "rag",
	Short: "RAG commands",
	Long:  "Ingest documents, query vector collections, and manage RAG storage.",
}

// initFlags initialises persistent and subcommand flags once.
func initFlags() {
	initFlagsOnce.Do(func() {
		// Qdrant connection flags (persistent) - defaults to localhost for local development
		qHost := "localhost"
		if v := core.Env("QDRANT_HOST"); v != "" {
			qHost = v
		}
		ragCmd.PersistentFlags().StringVar(&qdrantHost, "qdrant-host", qHost, "Qdrant host")

		qPort := envPortOrDefault("QDRANT_PORT", 6334)
		ragCmd.PersistentFlags().IntVar(&qdrantPort, "qdrant-port", qPort, "Qdrant gRPC port")

		// Ollama connection flags (persistent) - defaults to localhost for local development
		oHost := "localhost"
		if v := core.Env("OLLAMA_HOST"); v != "" {
			oHost = v
		}
		ragCmd.PersistentFlags().StringVar(&ollamaHost, "ollama-host", oHost, "Ollama host")

		oPort := envPortOrDefault("OLLAMA_PORT", 11434)
		ragCmd.PersistentFlags().IntVar(&ollamaPort, "ollama-port", oPort, "Ollama port")

		m := "nomic-embed-text"
		if v := core.Env("EMBEDDING_MODEL"); v != "" {
			m = v
		}
		ragCmd.PersistentFlags().StringVar(&model, "model", m, "Embedding model")

		// Verbose flag (persistent)
		ragCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

		// Ingest command flags
		ingestCmd.Flags().StringVar(&collection, "collection", "hostuk-docs", "Collection name")
		ingestCmd.Flags().BoolVar(&recreate, "recreate", false, "Recreate the collection before ingesting")
		ingestCmd.Flags().IntVar(&chunkSize, "chunk-size", 500, "Chunk size in characters")
		ingestCmd.Flags().IntVar(&chunkOverlap, "chunk-overlap", 50, "Chunk overlap in characters")

		// Query command flags
		queryCmd.Flags().StringVar(&queryCollection, "collection", "hostuk-docs", "Collection name")
		queryCmd.Flags().IntVar(&limit, "top", 5, "Maximum results")
		queryCmd.Flags().Float32Var(&threshold, "threshold", 0.5, "Minimum score threshold")
		queryCmd.Flags().StringVar(&category, "category", "", "Category filter")
		queryCmd.Flags().BoolVar(&keywords, "keywords", false, "Enable keyword fallback")
		queryCmd.Flags().StringVar(&format, "format", "text", "Output format: text, json, or context")

		// Collections command flags
		collectionsCmd.Flags().BoolVar(&listCollections, "list", false, "List collections")
		collectionsCmd.Flags().BoolVar(&showStats, "stats", false, "Show collection statistics")
		collectionsCmd.Flags().StringVar(&deleteCollection, "delete", "", "Delete a collection")
	})
}

// envPortOrDefault returns an environment port override or the fallback port.
func envPortOrDefault(name string, fallback int) int {
	value := core.Env(name)
	if value == "" {
		return fallback
	}
	port, err := strconv.Atoi(value)
	if err != nil {
		panic(core.E("rag.cmd.flags", core.Sprintf("invalid %s value: %s", name, value), err))
	}
	if port < 1 || port > 65535 {
		panic(core.E("rag.cmd.flags", core.Sprintf("invalid %s value: %d", name, port), nil))
	}
	return port
}
