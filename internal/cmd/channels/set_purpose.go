package channels

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type setPurposeOptions struct{}

func newSetPurposeCmd() *cobra.Command {
	opts := &setPurposeOptions{}

	return &cobra.Command{
		Use:   "set-purpose <channel-id> <purpose>",
		Short: "Set channel purpose",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetPurpose(args[0], args[1], opts, nil)
		},
	}
}

func runSetPurpose(channelID, purpose string, opts *setPurposeOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.SetChannelPurpose(channelID, purpose); err != nil {
		return err
	}

	output.Printf("Set purpose for channel %s\n", channelID)
	return nil
}
