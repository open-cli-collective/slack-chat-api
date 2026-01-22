package channels

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type listOptions struct {
	types           string
	excludeArchived bool
	limit           int
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.types, "types", "", "Channel types (public_channel,private_channel,mpim,im)")
	cmd.Flags().BoolVar(&opts.excludeArchived, "exclude-archived", true, "Exclude archived channels")
	cmd.Flags().IntVar(&opts.limit, "limit", 100, "Maximum channels to return")

	return cmd
}

func runList(opts *listOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channels, err := c.ListChannels(opts.types, opts.excludeArchived, opts.limit)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(channels)
	}

	if len(channels) == 0 {
		output.Println("No channels found")
		return nil
	}

	headers := []string{"ID", "NAME", "MEMBERS"}
	rows := make([][]string, 0, len(channels))
	for _, ch := range channels {
		members := fmt.Sprintf("%d", ch.NumMembers)
		if ch.IsPrivate {
			members += " (private)"
		}
		rows = append(rows, []string{ch.ID, ch.Name, members})
	}
	output.Table(headers, rows)

	return nil
}
