package search

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type messagesOptions struct {
	count       int
	page        int
	sort        string
	sortDir     string
	highlight   bool
	scope       string
	inChannel   string
	fromUser    string
	after       string
	before      string
	hasLink     bool
	hasReaction bool
}

func newMessagesCmd() *cobra.Command {
	opts := &messagesOptions{}

	cmd := &cobra.Command{
		Use:   "messages <query>",
		Short: "Search messages",
		Long: `Search messages across channels.

Requires a user token (xoxp-*) with search:read scope.

Search modifiers (can also use flags below):
  in:#channel    Search in specific channel
  in:@user       Search in DMs with user
  from:@user     Messages from specific user
  before:date    Messages before date (YYYY-MM-DD)
  after:date     Messages after date (YYYY-MM-DD)
  has:link       Messages containing links
  has:reaction   Messages with reactions

Examples:
  slck search messages "quarterly report"
  slck search messages "bug fix" --in "#engineering"
  slck search messages "project update" --from "@alice"
  slck search messages "deployment" --after 2025-01-01
  slck search messages "test" --scope public
  slck search messages "meeting" --has-link --has-reaction`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchMessages(args[0], opts, nil)
		},
	}

	cmd.Flags().IntVarP(&opts.count, "count", "c", 20, "Results per page (max 100)")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number (max 100)")
	cmd.Flags().StringVarP(&opts.sort, "sort", "s", "score", "Sort by: score or timestamp")
	cmd.Flags().StringVar(&opts.sortDir, "sort-dir", "desc", "Sort direction: asc or desc")
	cmd.Flags().BoolVar(&opts.highlight, "highlight", false, "Highlight matching terms in results")

	// Query builder flags
	cmd.Flags().StringVar(&opts.scope, "scope", "", "Search scope: all, public, private, dm, mpim")
	cmd.Flags().StringVar(&opts.inChannel, "in", "", "Filter by channel (e.g., \"#general\" or \"general\")")
	cmd.Flags().StringVar(&opts.fromUser, "from", "", "Filter by user (e.g., \"@alice\" or \"alice\")")
	cmd.Flags().StringVar(&opts.after, "after", "", "Messages after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.before, "before", "", "Messages before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&opts.hasLink, "has-link", false, "Messages containing links")
	cmd.Flags().BoolVar(&opts.hasReaction, "has-reaction", false, "Messages with reactions")

	return cmd
}

func runSearchMessages(query string, opts *messagesOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.NewUserClient()
		if err != nil {
			return err
		}
	}

	// Validate options
	if err := validateSearchOptions(opts.count, opts.page, opts.sort, opts.sortDir); err != nil {
		return err
	}

	// Validate and build query with options
	queryOpts := &QueryOptions{
		Scope:       opts.scope,
		InChannel:   opts.inChannel,
		FromUser:    opts.fromUser,
		After:       opts.after,
		Before:      opts.before,
		HasLink:     opts.hasLink,
		HasReaction: opts.hasReaction,
	}
	if err := ValidateQueryOptions(queryOpts); err != nil {
		return err
	}
	finalQuery := BuildQuery(query, queryOpts)

	result, err := c.SearchMessages(finalQuery, opts.count, opts.page, opts.sort, opts.sortDir, opts.highlight)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(result)
	}

	// Text/table output
	if result.Messages == nil || len(result.Messages.Matches) == 0 {
		output.Printf("No messages found for \"%s\"\n", query)
		return nil
	}

	output.Printf("Found %d messages matching \"%s\"\n\n", result.Messages.Total, query)

	headers := []string{"CHANNEL", "USER", "TIMESTAMP", "TEXT"}
	rows := make([][]string, 0, len(result.Messages.Matches))
	for _, m := range result.Messages.Matches {
		text := truncateText(m.Text, 60)
		ts := formatTimestamp(m.TS)
		rows = append(rows, []string{m.Channel.Name, m.Username, ts, text})
	}
	output.Table(headers, rows)

	paging := result.Messages.Paging
	output.Printf("\nPage %d of %d (showing %d of %d results)\n",
		paging.Page, paging.Pages, len(result.Messages.Matches), paging.Total)

	return nil
}

func validateSearchOptions(count, page int, sort, sortDir string) error {
	if count < 1 || count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}
	if page < 1 || page > 100 {
		return fmt.Errorf("page must be between 1 and 100")
	}
	if sort != "score" && sort != "timestamp" {
		return fmt.Errorf("sort must be 'score' or 'timestamp'")
	}
	if sortDir != "asc" && sortDir != "desc" {
		return fmt.Errorf("sort-dir must be 'asc' or 'desc'")
	}
	return nil
}

func truncateText(s string, maxLen int) string {
	// Remove newlines for cleaner table display
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatTimestamp(ts string) string {
	// Slack timestamps are Unix timestamps with microseconds (e.g., "1234567890.123456")
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return ts
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ts
	}

	t := time.Unix(sec, 0)
	return t.Format("2006-01-02 15:04")
}
