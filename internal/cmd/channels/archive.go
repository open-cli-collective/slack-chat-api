package channels

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type archiveOptions struct{}

func newArchiveCmd() *cobra.Command {
	opts := &archiveOptions{}

	return &cobra.Command{
		Use:   "archive <channel-id>",
		Short: "Archive a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(args[0], opts, nil)
		},
	}
}

func runArchive(channelID string, opts *archiveOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.ArchiveChannel(channelID); err != nil {
		return err
	}

	output.Printf("Archived channel: %s\n", channelID)
	return nil
}
