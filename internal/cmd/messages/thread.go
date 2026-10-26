package messages

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/render"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

type threadOptions struct {
	limit int
	since string
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
	cmd.Flags().StringVar(&opts.since, "since", "", "Only return messages after this timestamp")

	return cmd
}

func runThread(channel, threadTS string, opts *threadOptions, c *client.Client) error {
	// Normalize thread timestamp (accepts API format, p-prefixed, or full URL)
	threadTS = validate.NormalizeTimestamp(threadTS)

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	// Resolve channel name to ID if needed
	channelID, err := c.ResolveMessageDestination(channel)
	if err != nil {
		return err
	}

	messages, err := c.GetThreadReplies(channelID, threadTS, opts.limit, opts.since)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		output.Println("No replies found")
		return nil
	}

	renderMessageList(messages, client.NewUserResolver(c))
	return nil
}

// renderMessageList prints the messages as render.MessageList formats them.
func renderMessageList(messages []client.Message, resolver *client.UserResolver) {
	output.Printf("%s", render.MessageList(messages, resolver))
}
