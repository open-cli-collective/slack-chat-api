package search

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type filesOptions struct {
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
	fileType    string
	hasPin      bool
	includeBots bool
}

func newFilesCmd() *cobra.Command {
	opts := &filesOptions{}

	cmd := &cobra.Command{
		Use:   "files <query>",
		Short: "Search files",
		Long: `Search files across channels.

Requires a user token (xoxp-*) with search:read scope.

Search modifiers (can also use flags below):
  in:#channel    Search in specific channel
  from:@user     Files from specific user
  type:filetype  Filter by file type (pdf, doc, image, etc.)
  before:date    Files before date (YYYY-MM-DD)
  after:date     Files after date (YYYY-MM-DD)

Examples:
  slck search files "budget spreadsheet"
  slck search files "quarterly report" --in "#finance"
  slck search files "logo" --from "@alice" --type image
  slck search files "contract" --type pdf
  slck search files "document" --scope public`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearchFiles(args[0], opts, nil)
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
	cmd.Flags().StringVar(&opts.after, "after", "", "Files after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.before, "before", "", "Files before date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&opts.fileType, "type", "", "Filter by file type (pdf, doc, image, etc.)")
	cmd.Flags().BoolVar(&opts.hasPin, "has-pin", false, "Files that are pinned")
	cmd.Flags().BoolVar(&opts.includeBots, "include-bots", false, "Include bot messages in results")

	return cmd
}

func runSearchFiles(query string, opts *filesOptions, c *client.Client) error {
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
		Scope:     opts.scope,
		InChannel: opts.inChannel,
		FromUser:  opts.fromUser,
		After:     opts.after,
		Before:    opts.before,
		FileType:  opts.fileType,
		HasPin:    opts.hasPin,
	}
	if err := ValidateQueryOptions(queryOpts); err != nil {
		return err
	}
	finalQuery := BuildQuery(query, queryOpts)

	result, err := c.SearchFiles(finalQuery, opts.count, opts.page, opts.sort, opts.sortDir, opts.highlight, opts.includeBots)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(result)
	}

	// Text/table output
	if result.Files == nil || len(result.Files.Matches) == 0 {
		output.Printf("No files found for \"%s\"\n", query)
		return nil
	}

	output.Printf("Found %d files matching \"%s\"\n\n", result.Files.Total, query)

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
	output.Printf("\nPage %d of %d (showing %d of %d results)\n",
		paging.Page, paging.Pages, len(result.Files.Matches), paging.Total)

	return nil
}

func formatUnixTimestamp(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02")
}
