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

	// Resolve channel name to ID if needed
	channelID, err := c.ResolveChannel(channel)
	if err != nil {
		return err
	}

	messages, err := c.GetChannelHistory(channelID, opts.limit, opts.oldest, opts.latest)
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

	resolver := client.NewUserResolver(c)
	for _, m := range messages {
		ts := formatTimestamp(m.TS)
		// Compact view — truncation inherently flattens, so the
		// blocks-vs-text distinction doesn't matter here.
		body, _ := messageBody(m, resolver)
		text := truncate(body, 80)
		name := resolver.Resolve(m.User)
		edited := ""
		if m.Edited != nil {
			edited = " [edited]"
		}
		output.Printf("[%s] %s: %s%s\n", ts, name, text, edited)
		if files := renderFiles(m.Files); files != "" {
			output.Printf("%s", files)
		}
	}

	return nil
}
