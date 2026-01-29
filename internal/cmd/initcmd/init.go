package initcmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type initOptions struct {
	botToken  string
	userToken string
	noVerify  bool
	stdin     io.Reader                                  // For testing
	newClient func(baseURL, token string) *client.Client // For testing
}

// NewCmd creates the init command
func NewCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard",
		Long: `Set up slck with guided configuration.

This wizard walks you through configuring bot and user tokens
for Slack API access. Tokens are verified against the Slack API
unless --no-verify is passed.

For non-interactive use, provide tokens via flags:
  slck init --bot-token xoxb-... --user-token xoxp-...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.botToken, "bot-token", "", "Bot token (xoxb-*) for non-interactive setup")
	cmd.Flags().StringVar(&opts.userToken, "user-token", "", "User token (xoxp-*) for non-interactive setup")
	cmd.Flags().BoolVar(&opts.noVerify, "no-verify", false, "Skip token verification")

	return cmd
}

func (o *initOptions) reader() io.Reader {
	if o.stdin != nil {
		return o.stdin
	}
	return os.Stdin
}

func (o *initOptions) makeClient(token string) *client.Client {
	if o.newClient != nil {
		return o.newClient("", token)
	}
	return client.NewWithConfig("https://slack.com/api", token, nil)
}

func runInit(opts *initOptions) error {
	output.Println("Slack CLI Setup")
	output.Println()

	// Check for existing config
	hasBotToken := keychain.HasStoredToken()
	hasUserToken := keychain.HasStoredUserToken()
	if hasBotToken || hasUserToken {
		output.Println("Existing configuration detected.")
		if !promptYesNo(opts.reader(), "Overwrite existing configuration?", false) {
			output.Println("Setup cancelled.")
			return nil
		}
		output.Println()
	}

	output.Println("This CLI supports both bot tokens (xoxb-*) and user tokens (xoxp-*).")
	output.Println("Bot tokens are recommended for most use cases.")
	output.Println()

	// Bot token
	botToken := opts.botToken
	if botToken == "" {
		var err error
		botToken, err = promptToken(opts.reader(), "Bot Token (xoxb-...)")
		if err != nil {
			return err
		}
	}

	if botToken != "" {
		tokenType := keychain.DetectTokenType(botToken)
		if tokenType != "bot" {
			return fmt.Errorf("expected bot token (xoxb-*), got %s token", tokenType)
		}

		if !opts.noVerify {
			output.Println()
			output.Println("Testing connection...")
			c := opts.makeClient(botToken)
			info, err := c.AuthTest()
			if err != nil {
				return fmt.Errorf("bot token verification failed: %w", err)
			}
			output.Println("  Bot token valid")
			output.Printf("  Connected to workspace: %s\n", info.Team)
			output.Printf("  User: %s\n", info.User)
			if info.BotID != "" {
				output.Printf("  Bot ID: %s\n", info.BotID)
			}
		}

		if err := keychain.SetAPIToken(botToken); err != nil {
			return fmt.Errorf("failed to store bot token: %w", err)
		}
		output.Println()
		output.Println("Bot token saved.")
	}

	// User token
	userToken := opts.userToken
	if userToken == "" {
		output.Println()
		if promptYesNo(opts.reader(), "Would you like to add a user token as well? (needed for search)", false) {
			var err error
			userToken, err = promptToken(opts.reader(), "User Token (xoxp-...)")
			if err != nil {
				return err
			}
		}
	}

	if userToken != "" {
		tokenType := keychain.DetectTokenType(userToken)
		if tokenType != "user" {
			return fmt.Errorf("expected user token (xoxp-*), got %s token", tokenType)
		}

		if !opts.noVerify {
			output.Println()
			output.Println("Testing connection...")
			c := opts.makeClient(userToken)
			info, err := c.AuthTest()
			if err != nil {
				return fmt.Errorf("user token verification failed: %w", err)
			}
			output.Println("  User token valid")
			output.Printf("  Connected to workspace: %s\n", info.Team)
			output.Printf("  User: %s\n", info.User)
		}

		if err := keychain.SetUserToken(userToken); err != nil {
			return fmt.Errorf("failed to store user token: %w", err)
		}
		output.Println()
		output.Println("User token saved.")
	}

	if botToken == "" && userToken == "" {
		output.Println("No tokens provided. Setup cancelled.")
		return nil
	}

	output.Println()
	output.Println("Configuration saved. Try it out:")
	if botToken != "" {
		output.Println("  slck channels list")
		output.Println("  slck users list")
	}
	if userToken != "" {
		output.Println("  slck search messages \"hello\"")
	}

	return nil
}

func promptToken(reader io.Reader, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func promptYesNo(reader io.Reader, prompt string, defaultYes bool) bool {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	fmt.Printf("%s%s", prompt, suffix)

	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "" {
			return defaultYes
		}
		return answer == "y" || answer == "yes"
	}
	return defaultYes
}
