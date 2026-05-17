package config

import (
	"fmt"

	"github.com/spf13/cobra"
)

// set-token is hard-deprecated (§2.4 / §1.5). It accepted a secret as a
// positional argument — the worst ingress form (no `=` to spot in shell
// history, no hint the arg is sensitive). It now accepts the value via NO
// path (flag, positional, or stdin) and exits nonzero pointing at the
// sanctioned ingress, `slck set-credential`.
func newSetTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "set-token",
		Short:  "Removed — use 'slck set-credential' (see message)",
		Hidden: true,
		// Accept and ignore any args so we can emit our own guidance
		// rather than a generic cobra usage error.
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf(`'slck config set-token' has been removed.

It passed a secret as a positional argument, which leaks via shell history
and process listings. Use 'slck set-credential' instead, which reads the
value from stdin or a named env var only:

  slck set-credential --ref slack-chat-api/default --key bot_token --stdin
  op read 'op://Vault/Slack/bot_token' | slck set-credential --key bot_token --stdin

For full guided setup: slck init --bot-token-from-env SLACK_BOT_TOKEN`)
		},
	}
}
