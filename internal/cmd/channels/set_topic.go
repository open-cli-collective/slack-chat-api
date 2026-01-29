package channels

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type setTopicOptions struct{}

func newSetTopicCmd() *cobra.Command {
	opts := &setTopicOptions{}

	return &cobra.Command{
		Use:   "set-topic <channel> <topic>",
		Short: "Set channel topic",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetTopic(args[0], args[1], opts, nil)
		},
	}
}

func runSetTopic(channel, topic string, opts *setTopicOptions, c *client.Client) error {
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

	if err := c.SetChannelTopic(channelID, topic); err != nil {
		return err
	}

	output.Printf("Set topic for channel %s\n", channel)
	return nil
}
