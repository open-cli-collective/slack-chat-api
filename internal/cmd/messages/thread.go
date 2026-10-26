package messages

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
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

// renderMessageList prints FormatMessageList's output.
func renderMessageList(messages []client.Message, resolver *client.UserResolver) {
	output.Printf("%s", FormatMessageList(messages, resolver))
}

// FormatMessageList renders "[ts] user: body" lines for each message, with
// continuation indentation for multi-line bodies, edited markers, and file
// attachment and reaction lines. Shared by `messages thread`, `messages read`
// and the watch state `slck watch` keeps.
func FormatMessageList(messages []client.Message, resolver *client.UserResolver) string {
	var b strings.Builder
	for _, m := range messages {
		ts := formatTimestamp(m.TS)
		body, preserveNewlines := messageBody(m, resolver)
		var text string
		if preserveNewlines {
			text = indentContinuation(body)
		} else {
			text = flatten(body)
		}
		name := messageAuthor(m, resolver)
		edited := ""
		if m.Edited != nil {
			edited = " [edited]"
		}
		if edited != "" {
			if idx := strings.Index(text, "\n"); idx >= 0 {
				text = text[:idx] + edited + text[idx:]
				edited = ""
			}
		}
		fmt.Fprintf(&b, "[%s] %s: %s%s\n", ts, name, text, edited)
		b.WriteString(renderFiles(m.Files))
		b.WriteString(renderReactions(m.Reactions, resolver))
	}
	return b.String()
}
