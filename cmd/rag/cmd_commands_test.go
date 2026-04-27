package rag

import (
	"testing"

	"forge.lthn.ai/core/cli/pkg/cli"
	"github.com/stretchr/testify/assert"
)

func TestAddRAGSubcommands_Idempotent(t *testing.T) {
	parent := cli.NewGroup("root", "", "")

	AddRAGSubcommands(parent)
	AddRAGSubcommands(parent)

	children := parent.Commands()
	if assert.Len(t, children, 1) {
		assert.Equal(t, "rag", children[0].Name())
		assert.Len(t, children[0].Commands(), 3)
	}
}
