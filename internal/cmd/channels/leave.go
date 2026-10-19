package channels

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type leaveOptions struct{}

func newLeaveCmd() *cobra.Command {
	opts := &leaveOptions{}

	return &cobra.Command{
		Use:   "leave <channel>",
		Short: "Leave a channel as the token's user or bot",
		Long: `Leave a channel as the identity of the token in use. Leaving is
idempotent, and a public channel can be joined again with "slck channels join".`,
		Example: `  slck channels leave general --as-user`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLeave(args[0], opts, nil)
		},
	}
}

func runLeave(channel string, opts *leaveOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channelID, err := c.ResolveChannel(channel)
	if err != nil {
		return err
	}

	notMember, err := c.LeaveChannel(channelID)
	if err != nil {
		return client.WrapError(fmt.Sprintf("leave channel %s", channel), err)
	}
	if notMember {
		output.Printf("Not in channel %s\n", channel)
		return nil
	}
	output.Printf("Left channel %s\n", channel)
	return nil
}
