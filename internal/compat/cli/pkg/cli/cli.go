// SPDX-License-Identifier: EUPL-1.2

package cli

import "github.com/spf13/cobra"

// Command is the command type used by the RAG CLI.
type Command = cobra.Command

// PositionalArgs validates command positional arguments.
type PositionalArgs = cobra.PositionalArgs

// ExactArgs returns a positional validator requiring exactly n arguments.
func ExactArgs(n int) PositionalArgs {
	return cobra.ExactArgs(n)
}

// MaximumNArgs returns a positional validator allowing at most n arguments.
func MaximumNArgs(n int) PositionalArgs {
	return cobra.MaximumNArgs(n)
}

// NewGroup creates a parent command with no run handler.
func NewGroup(use string, short string, long string) *Command {
	cmd := &Command{
		Use:   use,
		Short: short,
	}
	if long != "" {
		cmd.Long = long
	}
	return cmd
}

// Style renders terminal text for command output.
type Style struct{}

// Render returns text unchanged for the compatibility CLI surface.
func (Style) Render(text string) string { return text }

var (
	// TitleStyle renders section titles.
	TitleStyle Style
	// ValueStyle renders named values.
	ValueStyle Style
	// ErrorStyle renders error values.
	ErrorStyle Style
	// DimStyle renders low-emphasis progress text.
	DimStyle Style
)
