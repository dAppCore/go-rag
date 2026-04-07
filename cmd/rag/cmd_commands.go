// Package rag provides RAG (Retrieval Augmented Generation) commands.
//
// Commands:
//   - core ai rag ingest: Ingest markdown files into Qdrant
//   - core ai rag query: Query the vector database
//   - core ai rag collections: List and manage collections
package rag

import (
	"dappco.re/go/core/cli/pkg/cli"
)

// AddRAGSubcommands registers the 'rag' command as a subcommand of parent.
// Called from the ai command package to mount under "core ai rag".
func AddRAGSubcommands(parent *cli.Command) {
	initFlags()
	ragCmd.AddCommand(ingestCmd)
	ragCmd.AddCommand(queryCmd)
	ragCmd.AddCommand(collectionsCmd)
	parent.AddCommand(ragCmd)
}
