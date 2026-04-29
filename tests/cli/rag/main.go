package main

import (
	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
	ragcmd "dappco.re/go/rag/cmd/rag"
)

func main() {
	root := cli.NewGroup("core", "RAG CLI artifact test driver", "")
	ai := cli.NewGroup("ai", "", "")
	root.AddCommand(ai)
	ragcmd.AddRAGSubcommands(ai)
	root.SetArgs(core.Args()[1:])

	if err := root.Execute(); err != nil {
		core.Print(core.Stderr(), "%v", err)
		core.Exit(1)
	}
}
