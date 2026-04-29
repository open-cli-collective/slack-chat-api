package messages

import (
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
	channelID, err := c.ResolveChannel(channel)
	if err != nil {
		return err
	}

	messages, err := c.GetThreadReplies(channelID, threadTS, opts.limit, opts.since)
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

	resolver := client.NewUserResolver(c)
	for _, m := range messages {
		ts := formatTimestamp(m.TS)
		body, preserveNewlines := messageBody(m, resolver)
		var text string
		if preserveNewlines {
			// Body came from a richer surface (blocks/attachments/files);
			// indent continuation lines so multi-line content stays
			// visually grouped under the [ts] user: header.
			text = indentContinuation(body)
		} else {
			// Plain-text fallback keeps existing single-line behavior.
			text = flatten(body)
		}
		name := resolver.Resolve(m.User)
		edited := ""
		if m.Edited != nil {
			edited = " [edited]"
		}
		// For multi-line bodies, place [edited] on the first line so it
		// annotates the whole message rather than appearing to annotate
		// only the final continuation line.
		if edited != "" {
			if idx := strings.Index(text, "\n"); idx >= 0 {
				text = text[:idx] + edited + text[idx:]
				edited = ""
			}
		}
		output.Printf("[%s] %s: %s%s\n", ts, name, text, edited)
		if files := renderFiles(m.Files); files != "" {
			output.Printf("%s", files)
		}
	}

	return nil
}
