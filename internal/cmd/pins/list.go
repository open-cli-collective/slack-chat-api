package pins

import (
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/render"
)

// pinTextMaxRunes is the width of the TEXT column, the one-line message width
// messages history uses.
const pinTextMaxRunes = 80

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <channel>",
		Short: "List the items pinned to a channel",
		Long: `List the items pinned to a channel, most recently pinned first.

Each row has the message's TS (what pins remove takes), when it was pinned,
who pinned it, its author and its text, cut to 80 characters. Requires the
pins:read scope.`,
		Example: `  slck pins list C1234567890
  slck pins list general --as-user`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runList(args[0], nil)
		},
	}
}

func runList(channel string, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channelID, err := c.ResolveMessageDestination(channel)
	if err != nil {
		return err
	}

	items, err := c.ListPins(channelID)
	if err != nil {
		return client.WrapScopeError("list pins", readScope, err)
	}

	if len(items) == 0 {
		output.Printf("No pins in %s\n", channel)
		return nil
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].Created > items[j].Created })

	resolver := client.NewUserResolver(c)
	headers := []string{"TS", "PINNED", "PINNED_BY", "AUTHOR", "TEXT"}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, pinRow(item, resolver))
	}
	output.SearchTable(headers, rows, pinTextMaxRunes)
	return nil
}

// pinRow renders one pinned item as a table row. A file pin has no message,
// so its TS and AUTHOR are "-" and its TEXT names the file.
func pinRow(item client.PinnedItem, resolver *client.UserResolver) []string {
	pinned := "-"
	if item.Created > 0 {
		pinned = render.Timestamp(strconv.FormatInt(item.Created, 10))
	}
	pinnedBy := orDash(resolver.Resolve(item.CreatedBy))

	if item.Message != nil {
		m := *item.Message
		body, _ := render.MessageBody(m, resolver)
		return []string{orDash(m.TS), pinned, pinnedBy, render.MessageAuthor(m, resolver), orDash(body)}
	}

	text := "[" + item.Type + "]"
	if f := item.File; f != nil {
		name := f.Title
		if name == "" {
			name = f.Name
		}
		if name == "" {
			name = f.ID
		}
		text += " " + name
	}
	return []string{"-", pinned, pinnedBy, "-", text}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
