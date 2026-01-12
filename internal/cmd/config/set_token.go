package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/keychain"
	"github.com/piekstra/slack-chat-api/internal/output"
)

type setTokenOptions struct{}

func newSetTokenCmd() *cobra.Command {
	opts := &setTokenOptions{}

	return &cobra.Command{
		Use:   "set-token [token]",
		Short: "Set a Slack API token",
		Long: `Set a Slack API token for authentication.

Token types are detected automatically:
  - Bot tokens (xoxb-*): Used for channels, users, messages commands
  - User tokens (xoxp-*): Used for search commands

On macOS: Tokens are stored securely in the system Keychain.
On Linux: Tokens are stored in ~/.config/slack-chat-api/credentials (file permissions 0600).

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
		output.Println("Warning: On Linux, your token will be stored in a config file")
		output.Println("         (~/.config/slack-chat-api/credentials) with restricted permissions (0600).")
		output.Println("         This is less secure than macOS Keychain storage.")
		output.Println()
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

	// Detect token type and store appropriately
	tokenType := keychain.DetectTokenType(token)

	switch tokenType {
	case "bot":
		if err := keychain.SetAPIToken(token); err != nil {
			return fmt.Errorf("failed to store bot token: %w", err)
		}
		if keychain.IsSecureStorage() {
			output.Println("Bot token stored securely in Keychain")
		} else {
			output.Println("Bot token stored in ~/.config/slack-chat-api/credentials")
		}
	case "user":
		if err := keychain.SetUserToken(token); err != nil {
			return fmt.Errorf("failed to store user token: %w", err)
		}
		if keychain.IsSecureStorage() {
			output.Println("User token stored securely in Keychain")
		} else {
			output.Println("User token stored in ~/.config/slack-chat-api/credentials")
		}
		output.Println("Note: This token will be used for search commands.")
	default:
		return fmt.Errorf("unrecognized token format (expected xoxb-* for bot or xoxp-* for user)")
	}

	return nil
}
