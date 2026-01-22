package users

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// ValidFields contains the allowed field filter values
var ValidFields = []string{"all", "name", "email", "display_name"}

type searchOptions struct {
	limit       int
	includeBots bool
	field       string
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search users by name, email, or display name",
		Long: `Search users by name, email, or display name.

Uses the users.list API with local filtering. Requires a bot token (xoxb-*)
with users:read scope.

By default, searches across all fields (username, real name, display name, email).
Use --field to limit search to a specific field.

Bot users are excluded by default. Use --include-bots to include them.

Examples:
  slack-chat-api users search "john"
  slack-chat-api users search "john@company.com" --field email
  slack-chat-api users search "John Smith" --field display_name
  slack-chat-api users search "bot" --include-bots`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(args[0], opts, nil)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 1000, "Maximum users to search through")
	cmd.Flags().BoolVar(&opts.includeBots, "include-bots", false, "Include bot users in results")
	cmd.Flags().StringVar(&opts.field, "field", "all", "Search field: all, name, email, display_name")

	return cmd
}

func validateField(field string) error {
	for _, f := range ValidFields {
		if field == f {
			return nil
		}
	}
	return fmt.Errorf("invalid field: %q (must be one of: %s)", field, strings.Join(ValidFields, ", "))
}

func runSearch(query string, opts *searchOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	// Validate field option
	if err := validateField(opts.field); err != nil {
		return err
	}

	// Fetch all users up to the limit
	users, err := c.ListUsers(opts.limit)
	if err != nil {
		return err
	}

	// Filter users based on query
	queryLower := strings.ToLower(query)
	var matches []client.User

	for _, u := range users {
		// Skip bots unless explicitly included
		if u.IsBot && !opts.includeBots {
			continue
		}

		if matchesQuery(u, queryLower, opts.field) {
			matches = append(matches, u)
		}
	}

	if output.IsJSON() {
		return output.PrintJSON(matches)
	}

	if len(matches) == 0 {
		output.Printf("No users found matching \"%s\"\n", query)
		return nil
	}

	output.Printf("Found %d users matching \"%s\"\n\n", len(matches), query)

	headers := []string{"ID", "USERNAME", "REAL NAME", "EMAIL"}
	rows := make([][]string, 0, len(matches))
	for _, u := range matches {
		rows = append(rows, []string{u.ID, u.Name, u.RealName, u.Profile.Email})
	}
	output.Table(headers, rows)

	return nil
}

func matchesQuery(u client.User, queryLower, field string) bool {
	switch field {
	case "name":
		return strings.Contains(strings.ToLower(u.Name), queryLower)
	case "email":
		return strings.Contains(strings.ToLower(u.Profile.Email), queryLower)
	case "display_name":
		return strings.Contains(strings.ToLower(u.Profile.DisplayName), queryLower) ||
			strings.Contains(strings.ToLower(u.RealName), queryLower)
	default: // "all"
		return strings.Contains(strings.ToLower(u.Name), queryLower) ||
			strings.Contains(strings.ToLower(u.RealName), queryLower) ||
			strings.Contains(strings.ToLower(u.Profile.DisplayName), queryLower) ||
			strings.Contains(strings.ToLower(u.Profile.Email), queryLower)
	}
}
