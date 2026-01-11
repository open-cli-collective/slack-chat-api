package users

import (
	"encoding/json"
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

	if output.JSON {
		data, _ := json.MarshalIndent(user, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("ID:           %s\n", user.ID)
	fmt.Printf("Username:     %s\n", user.Name)
	fmt.Printf("Real Name:    %s\n", user.RealName)
	fmt.Printf("Display Name: %s\n", user.Profile.DisplayName)
	fmt.Printf("Email:        %s\n", user.Profile.Email)
	fmt.Printf("Admin:        %t\n", user.IsAdmin)
	fmt.Printf("Bot:          %t\n", user.IsBot)
	if user.Profile.StatusText != "" {
		fmt.Printf("Status:       %s %s\n", user.Profile.StatusEmoji, user.Profile.StatusText)
	}

	return nil
}
