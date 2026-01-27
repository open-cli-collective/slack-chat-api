package config

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func newClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all stored credentials",
		Long: `Remove all stored Slack API tokens at once.

This is equivalent to running:
  slck config delete-token --type all --force

Note: Environment variables (SLACK_API_TOKEN, SLACK_USER_TOKEN) are not affected.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClear()
		},
	}
}

func runClear() error {
	clearedAny := false

	if keychain.HasStoredToken() {
		if err := keychain.DeleteAPIToken(); err != nil {
			return err
		}
		output.Println("Cleared bot token")
		clearedAny = true
	}

	if keychain.HasStoredUserToken() {
		if err := keychain.DeleteUserToken(); err != nil {
			return err
		}
		output.Println("Cleared user token")
		clearedAny = true
	}

	if !clearedAny {
		output.Println("No stored tokens to clear.")
	}

	// Warn about env vars
	hasEnvBot := keychain.GetTokenSource() == "environment variable"
	hasEnvUser := keychain.GetUserTokenSource() == "environment variable"
	if hasEnvBot || hasEnvUser {
		output.Println()
		output.Println("Note: Environment variable SLACK_API_TOKEN and/or SLACK_USER_TOKEN will still be used if set.")
	}

	return nil
}
