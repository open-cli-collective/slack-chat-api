package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/keychain"
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

func runShow(opts *showOptions) error {
	token, err := keychain.GetAPIToken()
	if err != nil {
		fmt.Println("API Token: Not configured")
		fmt.Println("\nRun 'slack-cli config set-token' to configure")
		return nil
	}

	// Mask the token for display
	masked := token[:8] + strings.Repeat("*", len(token)-12) + token[len(token)-4:]

	if keychain.IsSecureStorage() {
		fmt.Printf("API Token: %s (from Keychain)\n", masked)
	} else {
		fmt.Printf("API Token: %s (from config file)\n", masked)
	}

	return nil
}
