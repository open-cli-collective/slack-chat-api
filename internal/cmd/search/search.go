package search

import "github.com/spf13/cobra"

// NewCmd creates the search command and its subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search",
		Aliases: []string{"s"},
		Short:   "Search messages and files (requires user token)",
		Long: `Search Slack messages and files.

This command requires a user token (xoxp-*) with the search:read scope.
Bot tokens (xoxb-*) do not support search operations.

To set up a user token:
  1. Go to api.slack.com/apps -> Your app -> OAuth & Permissions
  2. Add 'search:read' to User Token Scopes
  3. Reinstall the app to your workspace
  4. Copy the User OAuth Token (starts with xoxp-)
  5. Run: slack-cli config set-token <your-xoxp-token>`,
	}

	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newFilesCmd())
	cmd.AddCommand(newAllCmd())

	return cmd
}
