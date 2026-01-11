package users

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type getOptions struct{}

func newGetCmd() *cobra.Command {
	opts := &getOptions{}

	return &cobra.Command{
		Use:   "get <user-id>",
		Short: "Get user information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(args[0], opts, nil)
		},
	}
}

func runGet(userID string, opts *getOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	user, err := c.GetUserInfo(userID)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(user)
	}

	output.KeyValue("ID", user.ID)
	output.KeyValue("Username", user.Name)
	output.KeyValue("Real Name", user.RealName)
	output.KeyValue("Display Name", user.Profile.DisplayName)
	output.KeyValue("Email", user.Profile.Email)
	output.KeyValue("Admin", user.IsAdmin)
	output.KeyValue("Bot", user.IsBot)
	if user.Profile.StatusText != "" {
		output.KeyValue("Status", fmt.Sprintf("%s %s", user.Profile.StatusEmoji, user.Profile.StatusText))
	}

	return nil
}
