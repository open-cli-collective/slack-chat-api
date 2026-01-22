package config

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type showOptions struct{}

func newShowCmd() *cobra.Command {
	opts := &showOptions{}

	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(opts)
		},
	}
}

func maskToken(token string) string {
	if len(token) < 12 {
		return strings.Repeat("*", len(token))
	}
	return token[:8] + strings.Repeat("*", len(token)-12) + token[len(token)-4:]
}

func runShow(opts *showOptions) error {
	hasAnyToken := false

	// Bot token
	botToken, botErr := keychain.GetAPIToken()
	if botErr == nil {
		hasAnyToken = true
		source := keychain.GetTokenSource()
		output.Printf("Bot Token: %s (from %s)\n", maskToken(botToken), source)
	} else {
		output.Println("Bot Token: Not configured")
	}

	// User token
	userToken, userErr := keychain.GetUserToken()
	if userErr == nil {
		hasAnyToken = true
		source := keychain.GetUserTokenSource()
		output.Printf("User Token: %s (from %s)\n", maskToken(userToken), source)
	} else {
		output.Println("User Token: Not configured (required for search)")
	}

	if !hasAnyToken {
		output.Println("\nRun 'slack-chat-api config set-token' to configure")
	}

	return nil
}
