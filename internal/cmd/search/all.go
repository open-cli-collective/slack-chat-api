package search

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type allOptions struct {
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

func newAllCmd() *cobra.Command {
	opts := &allOptions{}

	cmd := &cobra.Command{
		Use:   "all <query>",
		Short: "Search messages and files",
		Long: `Search both messages and files across channels.

Requires a user token (xoxp-*) with search:read scope.

This combines the results of search messages and search files.

Search modifiers (can also use flags below):
  in:#channel    Search in specific channel
  from:@user     Content from specific user
  before:date    Content before date (YYYY-MM-DD)
  after:date     Content after date (YYYY-MM-DD)

Examples:
  slck search all "project proposal"
  slck search all "quarterly report" --sort timestamp
  slck search all "important" --from "@alice"
  slck search all "meeting" --scope public
  slck search all "update" --after 2025-01-01 --in "#general"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchAll(args[0], opts, nil)
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
	cmd.Flags().StringVar(&opts.after, "after", "", "Content after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.before, "before", "", "Content before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&opts.hasLink, "has-link", false, "Content containing links")
	cmd.Flags().BoolVar(&opts.hasReaction, "has-reaction", false, "Content with reactions")

	return cmd
}

func runSearchAll(query string, opts *allOptions, c *client.Client) error {
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

	result, err := c.SearchAll(finalQuery, opts.count, opts.page, opts.sort, opts.sortDir, opts.highlight)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(result)
	}

	hasMessages := result.Messages != nil && len(result.Messages.Matches) > 0
	hasFiles := result.Files != nil && len(result.Files.Matches) > 0

	if !hasMessages && !hasFiles {
		output.Printf("No results found for \"%s\"\n", query)
		return nil
	}

	// Display messages section
	if hasMessages {
		output.Printf("=== Messages (%d total) ===\n\n", result.Messages.Total)

		headers := []string{"CHANNEL", "USER", "TIMESTAMP", "TEXT"}
		rows := make([][]string, 0, len(result.Messages.Matches))
		for _, m := range result.Messages.Matches {
			text := truncateText(m.Text, 60)
			ts := formatTimestamp(m.TS)
			rows = append(rows, []string{m.Channel.Name, m.Username, ts, text})
		}
		output.Table(headers, rows)

		paging := result.Messages.Paging
		output.Printf("\nPage %d of %d (showing %d of %d messages)\n",
			paging.Page, paging.Pages, len(result.Messages.Matches), paging.Total)
	}

	// Add spacing between sections
	if hasMessages && hasFiles {
		output.Println()
	}

	// Display files section
	if hasFiles {
		output.Printf("=== Files (%d total) ===\n\n", result.Files.Total)

		headers := []string{"NAME", "TYPE", "USER", "CREATED", "TITLE"}
		rows := make([][]string, 0, len(result.Files.Matches))
		for _, f := range result.Files.Matches {
			name := truncateText(f.Name, 30)
			title := truncateText(f.Title, 40)
			created := formatUnixTimestamp(f.Created)
			rows = append(rows, []string{name, f.Filetype, f.User, created, title})
		}
		output.Table(headers, rows)

		paging := result.Files.Paging
		output.Printf("\nPage %d of %d (showing %d of %d files)\n",
			paging.Page, paging.Pages, len(result.Files.Matches), paging.Total)
	}

	return nil
}
