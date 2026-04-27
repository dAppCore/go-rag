package main

import (
	"fmt"
	"os"

	"dappco.re/go/cli/pkg/cli"
	ragcmd "dappco.re/go/rag/cmd/rag"
)

func main() {
	root := cli.NewGroup("core", "RAG CLI artifact test driver", "")
	ai := cli.NewGroup("ai", "", "")
	root.AddCommand(ai)
	ragcmd.AddRAGSubcommands(ai)
	root.SetArgs(os.Args[1:])

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
