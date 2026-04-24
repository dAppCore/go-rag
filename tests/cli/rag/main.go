package main

import (
	"fmt"
	"os"

	ragcmd "dappco.re/go/rag/cmd/rag"
	"forge.lthn.ai/core/cli/pkg/cli"
)

func main() {
	root := cli.NewGroup("core-rag", "RAG CLI artifact test driver", "")
	ragcmd.AddRAGSubcommands(root)
	root.SetArgs(os.Args[1:])

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
