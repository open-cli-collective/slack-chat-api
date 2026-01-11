package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/keychain"
)

type deleteTokenOptions struct{}

func newDeleteTokenCmd() *cobra.Command {
	opts := &deleteTokenOptions{}

	return &cobra.Command{
		Use:   "delete-token",
		Short: "Delete the stored Slack API token",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteToken(opts)
		},
	}
}

func runDeleteToken(opts *deleteTokenOptions) error {
	if err := keychain.DeleteAPIToken(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	if keychain.IsSecureStorage() {
		fmt.Println("API token deleted from Keychain")
	} else {
		fmt.Println("API token deleted from config file")
	}
	return nil
}
