package messages

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

type permalinkOptions struct{}

func newPermalinkCmd() *cobra.Command {
	opts := &permalinkOptions{}

	return &cobra.Command{
		Use:   "permalink <channel> <timestamp>",
		Short: "Get a permalink URL to a message",
		Long: `Get a canonical permalink URL to a specific message.

Wraps Slack's chat.getPermalink, which returns the correct link for
top-level messages, thread replies, and enterprise-grid workspaces —
preferred over constructing an archive URL by hand.

  slck messages permalink C1234567890 1234567890.123456`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermalink(args[0], args[1], opts, nil)
		},
	}
}

func runPermalink(channel, timestamp string, _ *permalinkOptions, c *client.Client) error {
	if err := validate.Timestamp(timestamp); err != nil {
		return err
	}
	timestamp = validate.NormalizeTimestamp(timestamp)

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channelID, err := c.ResolveMessageDestination(channel)
	if err != nil {
		return err
	}

	permalink, err := c.GetPermalink(channelID, timestamp)
	if err != nil {
		return client.WrapError("get permalink", err)
	}

	output.Printf("%s\n", permalink)
	return nil
}
