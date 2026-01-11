package messages

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
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
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	// Remove colons if present
	emoji = strings.Trim(emoji, ":")

	if err := c.AddReaction(channel, timestamp, emoji); err != nil {
		return err
	}

	output.Printf("Added :%s: reaction\n", emoji)
	return nil
}
