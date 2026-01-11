package messages

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
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
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	// Remove colons if present
	emoji = strings.Trim(emoji, ":")

	if err := c.RemoveReaction(channel, timestamp, emoji); err != nil {
		return err
	}

	output.Printf("Removed :%s: reaction\n", emoji)
	return nil
}
