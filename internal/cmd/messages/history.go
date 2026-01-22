package messages

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type historyOptions struct {
	limit  int
	oldest string
	latest string
}

func newHistoryCmd() *cobra.Command {
	opts := &historyOptions{}

	cmd := &cobra.Command{
		Use:   "history <channel>",
		Short: "Get channel message history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(args[0], opts, nil)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 20, "Maximum messages to return")
	cmd.Flags().StringVar(&opts.oldest, "oldest", "", "Only messages after this timestamp")
	cmd.Flags().StringVar(&opts.latest, "latest", "", "Only messages before this timestamp")

	return cmd
}

func runHistory(channel string, opts *historyOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	messages, err := c.GetChannelHistory(channel, opts.limit, opts.oldest, opts.latest)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(messages)
	}

	if len(messages) == 0 {
		output.Println("No messages found")
		return nil
	}

	for _, m := range messages {
		ts := formatTimestamp(m.TS)
		text := truncate(m.Text, 80)
		output.Printf("[%s] %s: %s\n", ts, m.User, text)
	}

	return nil
}
