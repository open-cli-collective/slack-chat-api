package channels

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
)

type inviteOptions struct{}

func newInviteCmd() *cobra.Command {
	opts := &inviteOptions{}

	return &cobra.Command{
		Use:   "invite <channel-id> <user-id>...",
		Short: "Invite users to a channel",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvite(args[0], args[1:], opts, nil)
		},
	}
}

func runInvite(channelID string, userIDs []string, opts *inviteOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.InviteToChannel(channelID, userIDs); err != nil {
		return err
	}

	fmt.Printf("Invited %d user(s) to channel %s\n", len(userIDs), channelID)
	return nil
}
