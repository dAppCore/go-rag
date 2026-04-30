package main

import (
	core "dappco.re/go"
	ragcmd "dappco.re/go/rag/cmd/rag"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{Use: "core", Short: "RAG CLI artifact test driver"}
	ai := &cobra.Command{Use: "ai"}
	root.AddCommand(ai)
	ragcmd.AddRAGSubcommands(ai)
	root.SetArgs(core.Args()[1:])

	if err := root.Execute(); err != nil {
		core.Print(core.Stderr(), "%v", err)
		core.Exit(1)
	}
}
