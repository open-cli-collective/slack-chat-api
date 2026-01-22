package config

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type testOptions struct{}

func newTestCmd() *cobra.Command {
	opts := &testOptions{}

	return &cobra.Command{
		Use:   "test",
		Short: "Test Slack authentication",
		Long: `Verify that the configured API tokens authenticate successfully with Slack.

Tests both bot token (for most commands) and user token (for search commands).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(opts, nil, nil)
		},
	}
}

func runTest(opts *testOptions, botClient *client.Client, userClient *client.Client) error {
	output.Println("Testing Slack authentication...")
	output.Println()

	anySuccess := false

	// Test bot token
	output.Println("Bot Token:")
	if botClient == nil {
		var err error
		botClient, err = client.New()
		if err != nil {
			output.Printf("  Not configured: %v\n", err)
		}
	}
	if botClient != nil {
		info, err := botClient.AuthTest()
		if err != nil {
			output.Printf("  Authentication failed: %v\n", err)
		} else {
			anySuccess = true
			output.Println("  Authentication successful")
			output.Printf("    Workspace: %s\n", info.Team)
			output.Printf("    User: %s\n", info.User)
			if info.BotID != "" {
				output.Printf("    Bot ID: %s\n", info.BotID)
			}
		}
	}

	output.Println()

	// Test user token
	output.Println("User Token:")
	if userClient == nil {
		var err error
		userClient, err = client.NewUserClient()
		if err != nil {
			output.Printf("  Not configured: %v\n", err)
		}
	}
	if userClient != nil {
		info, err := userClient.AuthTest()
		if err != nil {
			output.Printf("  Authentication failed: %v\n", err)
		} else {
			anySuccess = true
			output.Println("  Authentication successful")
			output.Printf("    Workspace: %s\n", info.Team)
			output.Printf("    User: %s\n", info.User)
		}
	}

	if !anySuccess {
		output.Println()
		output.Println("No valid tokens configured. Run 'slack-chat-api config set-token' to configure.")
	}

	return nil
}
