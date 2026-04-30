package rag

import (
	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

func ExampleAddRAGSubcommands() {
	parent := cli.NewGroup("root", "", "")
	AddRAGSubcommands(parent)
	core.Println(len(parent.Commands()), parent.Commands()[0].Name())
	// Output: 1 rag
}
