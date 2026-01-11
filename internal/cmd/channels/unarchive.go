package channels

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type unarchiveOptions struct{}

func newUnarchiveCmd() *cobra.Command {
	opts := &unarchiveOptions{}

	return &cobra.Command{
		Use:   "unarchive <channel-id>",
		Short: "Unarchive a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnarchive(args[0], opts, nil)
		},
	}
}

func runUnarchive(channelID string, opts *unarchiveOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.UnarchiveChannel(channelID); err != nil {
		return err
	}

	output.Printf("Unarchived channel: %s\n", channelID)
	return nil
}
