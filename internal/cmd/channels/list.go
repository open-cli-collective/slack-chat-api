package channels

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
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

	if output.JSON {
		data, _ := json.MarshalIndent(channels, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(channels) == 0 {
		fmt.Println("No channels found")
		return nil
	}

	fmt.Printf("%-12s %-30s %s\n", "ID", "NAME", "MEMBERS")
	fmt.Println(strings.Repeat("-", 60))
	for _, ch := range channels {
		private := ""
		if ch.IsPrivate {
			private = " (private)"
		}
		fmt.Printf("%-12s %-30s %d%s\n", ch.ID, ch.Name, ch.NumMembers, private)
	}

	return nil
}
