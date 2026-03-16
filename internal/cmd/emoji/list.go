package emoji

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type listOptions struct {
	includeAliases bool
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List custom workspace emoji",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, nil)
		},
	}

	cmd.Flags().BoolVar(&opts.includeAliases, "include-aliases", false, "Include emoji aliases")

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

	emojis, err := c.ListEmoji()
	if err != nil {
		return err
	}

	// Filter aliases unless requested
	if !opts.includeAliases {
		filtered := make(map[string]string, len(emojis))
		for name, url := range emojis {
			if !strings.HasPrefix(url, "alias:") {
				filtered[name] = url
			}
		}
		emojis = filtered
	}

	// Sort names for consistent output
	names := make([]string, 0, len(emojis))
	for name := range emojis {
		names = append(names, name)
	}
	sort.Strings(names)

	if output.IsJSON() {
		return output.PrintJSON(emojis)
	}

	if len(names) == 0 {
		output.Println("No custom emoji found")
		return nil
	}

	for _, name := range names {
		output.Println(name)
	}

	return nil
}
