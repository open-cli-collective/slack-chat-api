package messages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

type unreactOptions struct{}

func newUnreactCmd() *cobra.Command {
	opts := &unreactOptions{}

	return &cobra.Command{
		Use:   "unreact <channel> <timestamp> <emoji>",
		Short: "Remove a reaction from a message",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnreact(args[0], args[1], args[2], opts, nil)
		},
	}
}

func runUnreact(channel, timestamp, emoji string, opts *unreactOptions, c *client.Client) error {
	// Validate inputs
	if err := validate.ChannelID(channel); err != nil {
		return err
	}
	if err := validate.Timestamp(timestamp); err != nil {
		return err
	}

	// Normalize emoji (remove colons)
	emoji = validate.Emoji(emoji)

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.RemoveReaction(channel, timestamp, emoji); err != nil {
		return client.WrapError(fmt.Sprintf("remove reaction :%s:", emoji), err)
	}

	output.Printf("Removed :%s: reaction\n", emoji)
	return nil
}
