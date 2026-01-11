package channels

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type setTopicOptions struct{}

func newSetTopicCmd() *cobra.Command {
	opts := &setTopicOptions{}

	return &cobra.Command{
		Use:   "set-topic <channel-id> <topic>",
		Short: "Set channel topic",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetTopic(args[0], args[1], opts, nil)
		},
	}
}

func runSetTopic(channelID, topic string, opts *setTopicOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.SetChannelTopic(channelID, topic); err != nil {
		return err
	}

	output.Printf("Set topic for channel %s\n", channelID)
	return nil
}
