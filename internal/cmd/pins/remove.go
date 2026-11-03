package pins

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <channel> <timestamp>",
		Short: "Unpin a message from a channel",
		Long: `Unpin a message from a channel. The message itself is not changed.

The timestamp takes the same forms as pins add; pins list prints it in the
TS column. Unpinning a message that is not pinned succeeds and says so.
Requires the pins:write scope.`,
		Example: `  slck pins remove C1234567890 1234567890.123456
  slck pins remove general 1234567890.123456 --as-user`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdutil.Quiet(cmd)
			return runRemove(args[0], args[1], nil)
		},
	}
}

func runRemove(channel, timestamp string, c *client.Client) error {
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

	notPinned, err := c.RemovePin(channelID, timestamp)
	if err != nil {
		return client.WrapScopeError("unpin message", writeScope, err)
	}
	if notPinned {
		output.Printf("Message %s is not pinned in %s\n", timestamp, channel)
		return nil
	}
	output.Printf("Unpinned message %s in %s\n", timestamp, channel)
	return nil
}
