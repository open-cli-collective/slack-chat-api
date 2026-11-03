package pins

import (
	"github.com/spf13/cobra"
)

// The OAuth scopes the pins methods need, on a bot or a user token alike.
const (
	readScope  = "pins:read"
	writeScope = "pins:write"
)

// NewCmd creates the pins command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pins",
		Aliases: []string{"pin"},
		Short:   "Manage pinned messages",
		Long: `Pin and unpin messages in a channel, and list what is pinned.

add and remove need the pins:write scope and list needs pins:read, on the bot
token, or on the user token with --as-user.`,
	}

	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newListCmd())

	return cmd
}

// quiet stops cobra from printing the usage text and its own copy of a
// runtime error once the arguments parsed, so the root command prints the
// error and its hint alone.
func quiet(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
}
