package channels

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type joinOptions struct{}

func newJoinCmd() *cobra.Command {
	opts := &joinOptions{}

	return &cobra.Command{
		Use:   "join <channel>",
		Short: "Join a public channel as the token's user or bot",
		Long: `Join a public channel as the identity of the token in use.

With --as-user this joins as you, which is what lets you invite others to a
channel you are not yet in. Joining is idempotent.`,
		Example: `  slck channels join general --as-user
  slck channels join C1234567890 --as-user`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJoin(args[0], opts, nil)
		},
	}
}

func runJoin(channel string, opts *joinOptions, c *client.Client) error {
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

	alreadyMember, err := c.JoinChannel(channelID)
	if err != nil {
		return client.WrapError(fmt.Sprintf("join channel %s", channel), err)
	}
	if alreadyMember {
		output.Printf("Already in channel %s\n", channel)
		return nil
	}
	output.Printf("Joined channel %s\n", channel)
	return nil
}
