package channels

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
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
		if strings.Contains(err.Error(), "not_in_channel") {
			output.Println("Error: Cannot unarchive channel.")
			output.Println("This is a Slack API limitation: bot tokens (xoxb-) cannot unarchive channels.")
			output.Println("Workaround: Use a user token (xoxp-) or unarchive via the Slack UI.")
			return err
		}
		return err
	}

	output.Printf("Unarchived channel: %s\n", channelID)
	return nil
}
