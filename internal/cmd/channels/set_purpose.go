package channels

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type setPurposeOptions struct{}

func newSetPurposeCmd() *cobra.Command {
	opts := &setPurposeOptions{}

	return &cobra.Command{
		Use:   "set-purpose <channel> <purpose>",
		Short: "Set channel purpose",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetPurpose(args[0], args[1], opts, nil)
		},
	}
}

func runSetPurpose(channel, purpose string, opts *setPurposeOptions, c *client.Client) error {
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

	if err := c.SetChannelPurpose(channelID, purpose); err != nil {
		return err
	}

	output.Printf("Set purpose for channel %s\n", channel)
	return nil
}
