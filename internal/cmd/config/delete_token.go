package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
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
		Short: "Delete stored Slack token(s) from the keyring",
		Long: `Delete stored Slack token(s) from the OS keyring.

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
	if opts.tokenType != "bot" && opts.tokenType != "user" && opts.tokenType != "all" {
		return fmt.Errorf("invalid token type: %s (must be bot, user, or all)", opts.tokenType)
	}

	st, err := keychain.Open()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	hasBot := st.HasBotToken()
	hasUser := st.HasUserToken()
	deleteBot := (opts.tokenType == "bot" || opts.tokenType == "all") && hasBot
	deleteUser := (opts.tokenType == "user" || opts.tokenType == "all") && hasUser

	if !deleteBot && !deleteUser {
		output.Println("No matching tokens stored to delete.")
		return nil
	}

	if !opts.force {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}

		var desc string
		switch {
		case deleteBot && deleteUser:
			desc = "bot and user tokens"
		case deleteBot:
			desc = "bot token"
		case deleteUser:
			desc = "user token"
		}

		output.Printf("About to delete the stored %s.\n", desc)
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

	if deleteBot {
		if err := st.DeleteBotToken(); err != nil {
			return fmt.Errorf("failed to delete bot token: %w", err)
		}
		output.Printf("Deleted bot_token from %s\n", st.Ref())
	}
	if deleteUser {
		if err := st.DeleteUserToken(); err != nil {
			return fmt.Errorf("failed to delete user token: %w", err)
		}
		output.Printf("Deleted user_token from %s\n", st.Ref())
	}
	return nil
}
