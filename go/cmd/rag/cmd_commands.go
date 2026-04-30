// Package rag provides RAG (Retrieval Augmented Generation) commands.
//
// Commands:
//   - core ai rag ingest: Ingest markdown files into Qdrant
//   - core ai rag query: Query the vector database
//   - core ai rag collections: List and manage collections
package rag

import (
	"sync"

	"dappco.re/go/cli/pkg/cli"
)

var addCommandsOnce sync.Once

// AddRAGSubcommands registers the 'rag' command as a subcommand of parent.
// Called from the ai command package to mount under "core ai rag".
// AddRAGSubcommands(rootCmd)
func AddRAGSubcommands(parent *cli.Command) {
	initFlags()

	addCommandsOnce.Do(func() {
		ragCmd.AddCommand(ingestCmd)
		ragCmd.AddCommand(queryCmd)
		ragCmd.AddCommand(collectionsCmd)
	})

	for _, cmd := range parent.Commands() {
		if cmd.Name() == ragCmd.Name() {
			return
		}
	}
	parent.AddCommand(ragCmd)
}
