package messages

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type threadOptions struct {
	limit int
}

func newThreadCmd() *cobra.Command {
	opts := &threadOptions{}

	cmd := &cobra.Command{
		Use:   "thread <channel> <thread-ts>",
		Short: "Get thread replies",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThread(args[0], args[1], opts, nil)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 100, "Maximum replies to return")

	return cmd
}

func runThread(channel, threadTS string, opts *threadOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	messages, err := c.GetThreadReplies(channel, threadTS, opts.limit)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(messages)
	}

	if len(messages) == 0 {
		output.Println("No replies found")
		return nil
	}

	for _, m := range messages {
		ts := formatTimestamp(m.TS)
		text := truncate(m.Text, 80)
		output.Printf("[%s] %s: %s\n", ts, m.User, text)
	}

	return nil
}
