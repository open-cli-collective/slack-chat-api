package unreads

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type listOptions struct {
	excludeChannels bool
	excludeDMs      bool
	includeApps     bool
}

type unreadConversation struct {
	id   string
	name string
}

// NewCmd creates the unreads command.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unreads",
		Short: "List unread conversations (requires user token)",
	}
	cmd.AddCommand(newListCmd())
	return cmd
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List unread channels and direct messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, nil)
		},
	}
	cmd.Flags().BoolVar(&opts.excludeChannels, "exclude-channels", false, "Exclude channels")
	cmd.Flags().BoolVar(&opts.excludeDMs, "exclude-dms", false, "Exclude direct messages")
	cmd.Flags().BoolVar(&opts.includeApps, "include-apps", false, "Include direct messages with agents and apps")
	return cmd
}

func runList(opts *listOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.NewUserClient()
		if err != nil {
			return err
		}
	}

	conversations, err := c.ListUserConversations()
	if err != nil {
		return err
	}

	users := map[string]client.User{}
	if !opts.excludeDMs || opts.includeApps {
		allUsers, err := c.ListAllUsers()
		if err != nil {
			return err
		}
		for _, user := range allUsers {
			users[user.ID] = user
		}
	}

	var channels, dms, apps []unreadConversation
	for _, conversation := range conversations {
		isApp := conversation.IsIM && isApp(users[conversation.User])
		switch {
		case conversation.IsIM || conversation.IsMpIM:
			if conversation.Priority <= 0 {
				continue
			}
			if (isApp && !opts.includeApps) || (!isApp && opts.excludeDMs) {
				continue
			}
		case opts.excludeChannels:
			continue
		}

		unread, err := isUnread(c, conversation.ID)
		if err != nil {
			return err
		}
		if !unread {
			continue
		}

		item := unreadConversation{id: conversation.ID, name: conversationName(conversation, users)}
		switch {
		case isApp:
			apps = append(apps, item)
		case conversation.IsIM || conversation.IsMpIM:
			dms = append(dms, item)
		default:
			channels = append(channels, item)
		}
	}

	if len(channels)+len(dms)+len(apps) == 0 {
		output.Println("No unread conversations")
		return nil
	}
	render("Channels", channels)
	render("Direct messages", dms)
	render("Agents & apps", apps)
	return nil
}

func isUnread(c *client.Client, conversationID string) (bool, error) {
	info, err := c.GetChannelInfo(conversationID)
	if err != nil {
		return false, fmt.Errorf("check unread status for %s: %w", conversationID, err)
	}
	if info.UnreadCount != nil {
		return *info.UnreadCount > 0, nil
	}

	oldest := info.LastRead
	if strings.Trim(oldest, "0.") == "" {
		oldest = ""
	}
	messages, err := c.GetChannelHistory(conversationID, 1, oldest, "")
	if err != nil {
		return false, fmt.Errorf("check unread status for %s: %w", conversationID, err)
	}
	return len(messages) > 0, nil
}

func isApp(user client.User) bool {
	return user.IsBot || user.IsAppUser || user.ID == "USLACKBOT"
}

func conversationName(conversation client.Channel, users map[string]client.User) string {
	if !conversation.IsIM {
		return conversation.Name
	}
	user, ok := users[conversation.User]
	if !ok {
		return conversation.User
	}
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.RealName != "" {
		return user.RealName
	}
	return user.Name
}

func render(title string, conversations []unreadConversation) {
	if len(conversations) == 0 {
		return
	}
	sort.Slice(conversations, func(i, j int) bool { return conversations[i].name < conversations[j].name })
	output.Println(title)
	rows := make([][]string, 0, len(conversations))
	for _, conversation := range conversations {
		rows = append(rows, []string{conversation.id, conversation.name})
	}
	output.Table([]string{"ID", "NAME"}, rows)
	output.Println()
}
