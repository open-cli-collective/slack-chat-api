package users

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
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

	if output.JSON {
		data, _ := json.MarshalIndent(users, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}

	fmt.Printf("%-12s %-20s %-30s %s\n", "ID", "USERNAME", "REAL NAME", "EMAIL")
	fmt.Println(strings.Repeat("-", 90))
	for _, u := range users {
		if u.IsBot {
			continue
		}
		fmt.Printf("%-12s %-20s %-30s %s\n", u.ID, u.Name, u.RealName, u.Profile.Email)
	}

	return nil
}
