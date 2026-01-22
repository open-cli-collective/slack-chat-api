package users

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type listOptions struct {
	limit int
}

func newListCmd() *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts, nil)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 100, "Maximum users to return")

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

	users, err := c.ListUsers(opts.limit)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(users)
	}

	if len(users) == 0 {
		output.Println("No users found")
		return nil
	}

	headers := []string{"ID", "USERNAME", "REAL NAME", "EMAIL"}
	rows := make([][]string, 0, len(users))
	for _, u := range users {
		if u.IsBot {
			continue
		}
		rows = append(rows, []string{u.ID, u.Name, u.RealName, u.Profile.Email})
	}
	output.Table(headers, rows)

	return nil
}
