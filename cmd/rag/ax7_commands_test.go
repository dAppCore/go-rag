package rag

import (
	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

func TestAX7_AddRAGSubcommands_Good(t *core.T) {
	parent := cli.NewGroup("root", "", "")
	AddRAGSubcommands(parent)

	core.AssertLen(t, parent.Commands(), 1)
	core.AssertEqual(t, "rag", parent.Commands()[0].Name())
}

func TestAX7_AddRAGSubcommands_Bad(t *core.T) {
	called := false
	core.AssertPanics(t, func() {
		called = true
		AddRAGSubcommands(nil)
	})

	core.AssertTrue(t, called)
	core.AssertNotNil(t, ragCmd)
}

func TestAX7_AddRAGSubcommands_Ugly(t *core.T) {
	parent := cli.NewGroup("root", "", "")
	AddRAGSubcommands(parent)
	AddRAGSubcommands(parent)

	core.AssertLen(t, parent.Commands(), 1)
	core.AssertLen(t, parent.Commands()[0].Commands(), 3)
}
