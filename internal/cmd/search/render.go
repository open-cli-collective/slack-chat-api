package search

import (
	"fmt"
	"io"
	"os"

	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// searchTextMaxRunes is the width of the last column in the default search
// table; --full prints that column in full instead.
const searchTextMaxRunes = 60

// fullNoticeThreshold is the number of full-text results above which --full
// prints a size notice. Full text is unbounded per result, so a page of them
// can be long to read and costly for a caller that feeds the output to a
// model.
const fullNoticeThreshold = 10

// noticeWriter receives the --full size notice; stderr keeps it out of the
// results a caller parses.
var noticeWriter io.Writer = os.Stderr

// renderSearchRows prints search results as the default one-line table, or,
// with full, as one block per result with the last column untruncated.
func renderSearchRows(headers []string, rows [][]string, full bool) {
	if full {
		if len(rows) > fullNoticeThreshold {
			_, _ = fmt.Fprintf(noticeWriter,
				"note: --full printed %d results in full; output can be large, so narrow the query or pass a smaller --count to limit it\n",
				len(rows))
		}
		output.SearchBlocks(headers, rows)
		return
	}
	output.SearchTable(headers, rows, searchTextMaxRunes)
}
