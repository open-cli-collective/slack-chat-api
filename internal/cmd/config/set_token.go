package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/keychain"
)

type setTokenOptions struct{}

func newSetTokenCmd() *cobra.Command {
	opts := &setTokenOptions{}

	return &cobra.Command{
		Use:   "set-token [token]",
		Short: "Set the Slack API token",
		Long: `Set the Slack API token for authentication.

On macOS: Token is stored securely in the system Keychain.
On Linux: Token is stored in ~/.config/slack-cli/credentials (file permissions 0600).

If no token is provided as an argument, you will be prompted to enter it.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var token string
			if len(args) > 0 {
				token = args[0]
			}
			return runSetToken(token, opts)
		},
	}
}

func runSetToken(token string, opts *setTokenOptions) error {
	// Warn Linux users about file-based storage
	if !keychain.IsSecureStorage() {
		fmt.Println("Warning: On Linux, your token will be stored in a config file")
		fmt.Println("         (~/.config/slack-cli/credentials) with restricted permissions (0600).")
		fmt.Println("         This is less secure than macOS Keychain storage.")
		fmt.Println()
	}

	if token == "" {
		fmt.Print("Enter Slack API token: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		token = strings.TrimSpace(input)
	}

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := keychain.SetAPIToken(token); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	if keychain.IsSecureStorage() {
		fmt.Println("API token stored securely in Keychain")
	} else {
		fmt.Println("API token stored in ~/.config/slack-cli/credentials")
	}
	return nil
}
