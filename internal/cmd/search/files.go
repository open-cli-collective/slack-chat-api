package search

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/client"
	"github.com/piekstra/slack-chat-api/internal/output"
)

type filesOptions struct {
	count     int
	page      int
	sort      string
	sortDir   string
	highlight bool
}

func newFilesCmd() *cobra.Command {
	opts := &filesOptions{}

	cmd := &cobra.Command{
		Use:   "files <query>",
		Short: "Search files",
		Long: `Search files across channels.

Requires a user token (xoxp-*) with search:read scope.

Search modifiers:
  in:#channel    Search in specific channel
  from:@user     Files from specific user
  type:filetype  Filter by file type (pdf, doc, image, etc.)
  before:date    Files before date (YYYY-MM-DD)
  after:date     Files after date (YYYY-MM-DD)

Examples:
  slack-chat-api search files "budget spreadsheet"
  slack-chat-api search files "in:#finance quarterly report"
  slack-chat-api search files "from:@alice type:pdf"
  slack-chat-api search files "type:image logo"`,
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

	result, err := c.SearchFiles(query, opts.count, opts.page, opts.sort, opts.sortDir, opts.highlight)
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
