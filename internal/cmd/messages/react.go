package messages

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/client"
	"github.com/piekstra/slack-chat-api/internal/output"
	"github.com/piekstra/slack-chat-api/internal/validate"
)

type reactOptions struct{}

func newReactCmd() *cobra.Command {
	opts := &reactOptions{}

	return &cobra.Command{
		Use:   "react <channel> <timestamp> <emoji>",
		Short: "Add a reaction to a message",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReact(args[0], args[1], args[2], opts, nil)
		},
	}
}

func runReact(channel, timestamp, emoji string, opts *reactOptions, c *client.Client) error {
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

	if err := c.AddReaction(channel, timestamp, emoji); err != nil {
		return client.WrapError(fmt.Sprintf("add reaction :%s:", emoji), err)
	}

	output.Printf("Added :%s: reaction\n", emoji)
	return nil
}
