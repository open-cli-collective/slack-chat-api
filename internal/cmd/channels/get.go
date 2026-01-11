package channels

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type getOptions struct{}

func newGetCmd() *cobra.Command {
	opts := &getOptions{}

	return &cobra.Command{
		Use:   "get <channel-id>",
		Short: "Get channel information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(args[0], opts, nil)
		},
	}
}

func runGet(channelID string, opts *getOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channel, err := c.GetChannelInfo(channelID)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(channel)
	}

	output.KeyValue("ID", channel.ID)
	output.KeyValue("Name", channel.Name)
	output.KeyValue("Private", channel.IsPrivate)
	output.KeyValue("Archived", channel.IsArchived)
	output.KeyValue("Members", channel.NumMembers)
	if channel.Topic.Value != "" {
		output.KeyValue("Topic", channel.Topic.Value)
	}
	if channel.Purpose.Value != "" {
		output.KeyValue("Purpose", channel.Purpose.Value)
	}

	return nil
}
