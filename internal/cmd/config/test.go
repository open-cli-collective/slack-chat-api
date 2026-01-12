package config

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type testOptions struct{}

func newTestCmd() *cobra.Command {
	opts := &testOptions{}

	return &cobra.Command{
		Use:   "test",
		Short: "Test Slack authentication",
		Long:  "Verify that the configured API token authenticates successfully with Slack.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(opts, nil)
		},
	}
}

func runTest(opts *testOptions, c *client.Client) error {
	output.Println("Testing Slack authentication...")

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	info, err := c.AuthTest()
	if err != nil {
		output.Println("Authentication failed")
		return err
	}

	output.Println("Authentication successful")
	output.KeyValue("Workspace", info.Team)
	output.KeyValue("User", info.User)
	if info.BotID != "" {
		output.KeyValue("Bot ID", info.BotID)
	}

	return nil
}
