package rag

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/cli/pkg/cli"
)

func TestAddRAGSubcommands_Idempotent(t *testing.T) {
	parent := cli.NewGroup("root", "", "")

	AddRAGSubcommands(parent)
	AddRAGSubcommands(parent)

	children := parent.Commands()
	if len(children) != 1 {
		t.Fatalf("want length %d, got %d", 1, len(children))
	}
	if children[0].Name() != "rag" {
		t.Fatalf("want %v, got %v", "rag", children[0].Name())
	}
	if got := len(children[0].Commands()); got != 3 {
		t.Fatalf("want length %d, got %d", 3, got)
	}
}

func TestCmdCommands_AddRAGSubcommands_Good(t *core.T) {
	parent := cli.NewGroup("root", "", "")
	AddRAGSubcommands(parent)

	core.AssertLen(t, parent.Commands(), 1)
	core.AssertEqual(t, "rag", parent.Commands()[0].Name())
}

func TestCmdCommands_AddRAGSubcommands_Bad(t *core.T) {
	called := false
	core.AssertPanics(t, func() {
		called = true
		AddRAGSubcommands(nil)
	})

	core.AssertTrue(t, called)
	core.AssertNotNil(t, ragCmd)
}

func TestCmdCommands_AddRAGSubcommands_Ugly(t *core.T) {
	parent := cli.NewGroup("root", "", "")
	AddRAGSubcommands(parent)
	AddRAGSubcommands(parent)

	core.AssertLen(t, parent.Commands(), 1)
	core.AssertLen(t, parent.Commands()[0].Commands(), 3)
}
