package channels

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type getOptions struct{}

func newGetCmd() *cobra.Command {
	opts := &getOptions{}

	return &cobra.Command{
		Use:   "get <channel>",
		Short: "Get channel information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(args[0], opts, nil)
		},
	}
}

func runGet(channel string, opts *getOptions, c *client.Client) error {
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

	ch, err := c.GetChannelInfo(channelID)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(ch)
	}

	output.KeyValue("ID", ch.ID)
	output.KeyValue("Name", ch.Name)
	output.KeyValue("Private", ch.IsPrivate)
	output.KeyValue("Archived", ch.IsArchived)
	output.KeyValue("Members", ch.NumMembers)
	if ch.Topic.Value != "" {
		output.KeyValue("Topic", ch.Topic.Value)
	}
	if ch.Purpose.Value != "" {
		output.KeyValue("Purpose", ch.Purpose.Value)
	}

	return nil
}
