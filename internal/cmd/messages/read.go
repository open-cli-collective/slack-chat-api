package messages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/messageref"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type readOptions struct {
	limit int
}

func newReadCmd() *cobra.Command {
	opts := &readOptions{}

	cmd := &cobra.Command{
		Use:   "read <message-ref>",
		Short: "Read a message thread by ref",
		Long: `Read the conversation thread for a message ref.

A message ref is the composite identity for a message in a channel:
  <channel_id>/<ts>           e.g. C02DF3BEUGN/1777469221.721439
  Slack permalink             e.g. https://workspace.slack.com/archives/C02DF3BEUGN/p1777469221721439

Refs are emitted as the REF column by 'slck search messages' and 'slck search all'.

Returns whatever Slack's conversations.replies returns for the ref. When the ref
points at a thread parent, that's the parent plus all replies. Some channels
require user-token semantics; pair with --as-user when search returned the ref.

Examples:
  slck messages read C02DF3BEUGN/1777469221.721439
  slck --as-user messages read C02DF3BEUGN/1777469221.721439`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRead(args[0], opts, nil)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 100, "Maximum messages to return")

	return cmd
}

func runRead(input string, opts *readOptions, c *client.Client) error {
	ref, err := messageref.Parse(input)
	if err != nil {
		return err
	}

	if c == nil {
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	messages, err := c.GetThreadReplies(ref.ChannelID, ref.TS, opts.limit, "")
	if err != nil {
		if client.IsSlackError(err, "not_in_channel") {
			return fmt.Errorf("%w\nHint: try `slck --as-user messages read %s` — search-derived refs typically require user-token access", err, ref)
		}
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(messages)
	}

	if len(messages) == 0 {
		output.Println("No messages found for ref")
		return nil
	}

	renderMessageList(messages, client.NewUserResolver(c))
	return nil
}
