package rag

import (
	"testing"

	"forge.lthn.ai/core/cli/pkg/cli"
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
