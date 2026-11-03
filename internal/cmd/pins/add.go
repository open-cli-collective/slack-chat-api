package pins

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <channel> <timestamp>",
		Short: "Pin a message to a channel",
		Long: `Pin a message to a channel.

The timestamp is the message's ts (1234567890.123456), its p-prefixed form,
or the message's Slack URL. Pinning a message that is already pinned
succeeds and says so. Requires the pins:write scope.`,
		Example: `  slck pins add C1234567890 1234567890.123456
  slck pins add general 1234567890.123456 --as-user`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.Quiet(cmd)
			return runAdd(args[0], args[1], nil)
		},
	}
}

func runAdd(channel, timestamp string, c *client.Client) error {
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

	alreadyPinned, err := c.AddPin(channelID, timestamp)
	if err != nil {
		return client.WrapScopeError("pin message", writeScope, err)
	}
	if alreadyPinned {
		output.Printf("Message %s is already pinned in %s\n", timestamp, channel)
		return nil
	}
	output.Printf("Pinned message %s in %s\n", timestamp, channel)
	return nil
}
