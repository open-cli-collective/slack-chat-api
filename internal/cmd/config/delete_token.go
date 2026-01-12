package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/keychain"
	"github.com/piekstra/slack-cli/internal/output"
)

type deleteTokenOptions struct {
	force     bool
	tokenType string
	stdin     io.Reader // For testing
}

func newDeleteTokenCmd() *cobra.Command {
	opts := &deleteTokenOptions{}

	cmd := &cobra.Command{
		Use:   "delete-token",
		Short: "Delete stored Slack API token(s)",
		Long: `Delete stored Slack API token(s).

Use --type to specify which token to delete:
  - bot: Delete the bot token (xoxb-*)
  - user: Delete the user token (xoxp-*)
  - all: Delete both tokens (default)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteToken(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")
	cmd.Flags().StringVarP(&opts.tokenType, "type", "t", "all", "Token type to delete: bot, user, or all")

	return cmd
}

func runDeleteToken(opts *deleteTokenOptions) error {
	// Validate token type
	if opts.tokenType != "bot" && opts.tokenType != "user" && opts.tokenType != "all" {
		return fmt.Errorf("invalid token type: %s (must be bot, user, or all)", opts.tokenType)
	}

	// Check what tokens exist
	hasBotToken := keychain.HasStoredToken()
	hasUserToken := keychain.HasStoredUserToken()

	// Determine what we're deleting
	deleteBot := (opts.tokenType == "bot" || opts.tokenType == "all") && hasBotToken
	deleteUser := (opts.tokenType == "user" || opts.tokenType == "all") && hasUserToken

	if !deleteBot && !deleteUser {
		switch opts.tokenType {
		case "bot":
			output.Println("No bot token stored to delete.")
			if os.Getenv("SLACK_API_TOKEN") != "" {
				output.Println("Note: Bot token is set via SLACK_API_TOKEN environment variable.")
			}
		case "user":
			output.Println("No user token stored to delete.")
			if os.Getenv("SLACK_USER_TOKEN") != "" {
				output.Println("Note: User token is set via SLACK_USER_TOKEN environment variable.")
			}
		default:
			output.Println("No tokens stored to delete.")
			if os.Getenv("SLACK_API_TOKEN") != "" || os.Getenv("SLACK_USER_TOKEN") != "" {
				output.Println("Note: Tokens may be set via environment variables.")
			}
		}
		return nil
	}

	// Prompt for confirmation unless --force
	if !opts.force {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}

		var tokenDesc string
		switch {
		case deleteBot && deleteUser:
			tokenDesc = "bot and user tokens"
		case deleteBot:
			tokenDesc = "bot token"
		case deleteUser:
			tokenDesc = "user token"
		}

		output.Printf("About to delete the stored %s.\n", tokenDesc)
		output.Printf("Are you sure? [y/N]: ")

		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			confirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if confirm != "y" && confirm != "yes" {
				output.Println("Cancelled.")
				return nil
			}
		}
	}

	// Delete tokens
	if deleteBot {
		if err := keychain.DeleteAPIToken(); err != nil {
			return fmt.Errorf("failed to delete bot token: %w", err)
		}
		if keychain.IsSecureStorage() {
			output.Println("Bot token deleted from Keychain")
		} else {
			output.Println("Bot token deleted from config file")
		}
	}

	if deleteUser {
		if err := keychain.DeleteUserToken(); err != nil {
			return fmt.Errorf("failed to delete user token: %w", err)
		}
		if keychain.IsSecureStorage() {
			output.Println("User token deleted from Keychain")
		} else {
			output.Println("User token deleted from config file")
		}
	}

	return nil
}
