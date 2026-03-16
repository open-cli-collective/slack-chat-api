package emoji

import (
	"github.com/spf13/cobra"
)

// NewCmd creates the emoji command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "emoji",
		Aliases: []string{"e"},
		Short:   "Manage Slack emoji",
	}

	cmd.AddCommand(newListCmd())

	return cmd
}
