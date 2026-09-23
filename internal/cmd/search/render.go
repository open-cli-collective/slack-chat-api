package search

import "github.com/open-cli-collective/slack-chat-api/internal/output"

// searchTextMaxRunes is the width of the last column in the default search
// table; --full prints that column in full instead.
const searchTextMaxRunes = 60

// renderSearchRows prints search results as the default one-line table, or,
// with full, as one block per result with the last column untruncated.
func renderSearchRows(headers []string, rows [][]string, full bool) {
	if full {
		output.SearchBlocks(headers, rows)
		return
	}
	output.SearchTable(headers, rows, searchTextMaxRunes)
}
