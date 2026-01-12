package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/cmd/channels"
	"github.com/piekstra/slack-chat-api/internal/cmd/config"
	"github.com/piekstra/slack-chat-api/internal/cmd/messages"
	"github.com/piekstra/slack-chat-api/internal/cmd/search"
	"github.com/piekstra/slack-chat-api/internal/cmd/users"
	"github.com/piekstra/slack-chat-api/internal/cmd/workspace"
	"github.com/piekstra/slack-chat-api/internal/output"
	"github.com/piekstra/slack-chat-api/internal/version"
)

var outputFormat string

var rootCmd = &cobra.Command{
	Use:   "slack-chat-api",
	Short: "A CLI tool for interacting with Slack",
	Long: `slack-chat-api is a command-line interface for Slack.

It provides commands for managing channels, users, messages,
and other Slack workspace operations.

Configure your API token with:
  slack-chat-api config set-token <your-token>

Or set the SLACK_API_TOKEN environment variable.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse and validate output format
		format, err := output.ParseFormat(outputFormat)
		if err != nil {
			return err
		}
		output.OutputFormat = format
		return nil
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, or table")
	rootCmd.PersistentFlags().BoolVar(&output.NoColor, "no-color", false, "Disable colored output")

	// Set custom version template to include commit and build date
	rootCmd.SetVersionTemplate("slack-chat-api " + version.Info() + "\n")

	// Add subcommands
	rootCmd.AddCommand(channels.NewCmd())
	rootCmd.AddCommand(users.NewCmd())
	rootCmd.AddCommand(messages.NewCmd())
	rootCmd.AddCommand(search.NewCmd())
	rootCmd.AddCommand(workspace.NewCmd())
	rootCmd.AddCommand(config.NewCmd())
}
