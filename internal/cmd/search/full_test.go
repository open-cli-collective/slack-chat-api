package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// longMultiLineText is longer than the default table column and spans lines,
// so the tests can tell the full rendering from the truncated one.
const longMultiLineText = "First line of a long message that goes well past the sixty character table column.\nSecond line.\nThird line with the ending marker END-OF-MESSAGE."

func messagesResponse(text string) map[string]interface{} {
	return map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total":  1,
			"paging": map[string]interface{}{"count": 20, "total": 1, "page": 1, "pages": 1},
			"matches": []map[string]interface{}{{
				"type":      "message",
				"channel":   map[string]interface{}{"id": "C1", "name": "general"},
				"user":      "U1",
				"username":  "alice",
				"text":      text,
				"ts":        "1704067200.000000",
				"permalink": "https://slack.com/archives/C1/p1",
			}},
		},
	}
}

func TestRunSearchMessages_FullPrintsCompleteMultiLineText(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(messagesResponse(longMultiLineText))
	})
	defer server.Close()

	opts := &messagesOptions{count: 20, page: 1, sort: "score", sortDir: "desc", full: true}
	out := captureOutput(t, func() {
		require.NoError(t, runSearchMessages("query", opts, c))
	})

	assert.Contains(t, out, "REF | CHANNEL | USER | WHEN\n")
	assert.Contains(t, out, "C1/1704067200.000000 | general | alice | ")
	assert.Contains(t, out, "\n  First line of a long message that goes well past the sixty character table column.\n")
	assert.Contains(t, out, "\n  Second line.\n")
	assert.Contains(t, out, "\n  Third line with the ending marker END-OF-MESSAGE.\n")
	assert.NotContains(t, out, "| TEXT")
}

func TestRunSearchMessages_DefaultOutputStillTruncates(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(messagesResponse(longMultiLineText))
	})
	defer server.Close()

	opts := &messagesOptions{count: 20, page: 1, sort: "score", sortDir: "desc"}
	out := captureOutput(t, func() {
		require.NoError(t, runSearchMessages("query", opts, c))
	})

	assert.Contains(t, out, "REF | CHANNEL | USER | WHEN | TEXT\n")
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, "END-OF-MESSAGE")
}

func TestRunSearchAll_FullPrintsCompleteMessageText(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(messagesResponse(longMultiLineText))
	})
	defer server.Close()

	opts := &allOptions{count: 20, page: 1, sort: "score", sortDir: "desc", full: true}
	out := captureOutput(t, func() {
		require.NoError(t, runSearchAll("query", opts, c))
	})

	assert.Contains(t, out, "=== Messages (1 total) ===")
	assert.Contains(t, out, "\n  Third line with the ending marker END-OF-MESSAGE.\n")
}

func TestRunSearchFiles_FullPrintsCompleteName(t *testing.T) {
	longTitle := strings.Repeat("Long file title ", 6) + "END-OF-TITLE"
	response := map[string]interface{}{
		"ok": true,
		"files": map[string]interface{}{
			"total":  1,
			"paging": map[string]interface{}{"count": 20, "total": 1, "page": 1, "pages": 1},
			"matches": []map[string]interface{}{{
				"id":       "F1",
				"name":     "notes.txt",
				"title":    longTitle,
				"filetype": "text",
				"user":     "U1",
				"created":  1704067200,
			}},
		},
	}
	c, server := newTestClient(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	full := captureOutput(t, func() {
		require.NoError(t, runSearchFiles("notes", &filesOptions{count: 20, page: 1, sort: "score", sortDir: "desc", full: true}, c))
	})
	assert.Contains(t, full, "END-OF-TITLE")

	table := captureOutput(t, func() {
		require.NoError(t, runSearchFiles("notes", &filesOptions{count: 20, page: 1, sort: "score", sortDir: "desc"}, c))
	})
	assert.NotContains(t, table, "END-OF-TITLE")
}

func TestFullNoticeAboveThreshold(t *testing.T) {
	var notice bytes.Buffer
	orig := noticeWriter
	noticeWriter = &notice
	defer func() { noticeWriter = orig }()

	var out bytes.Buffer
	origOut := output.Writer
	output.Writer = &out
	defer func() { output.Writer = origOut }()

	rows := make([][]string, fullNoticeThreshold)
	for i := range rows {
		rows[i] = []string{"C1/1.0", "text"}
	}
	renderSearchRows([]string{"REF", "TEXT"}, rows, true)
	if notice.Len() != 0 {
		t.Errorf("expected no notice at the threshold, got %q", notice.String())
	}

	rows = append(rows, []string{"C1/2.0", "text"})
	renderSearchRows([]string{"REF", "TEXT"}, rows, true)
	if !strings.Contains(notice.String(), "11 results") {
		t.Errorf("expected a notice above the threshold, got %q", notice.String())
	}
	if strings.Contains(out.String(), "note:") {
		t.Errorf("notice leaked into the results output")
	}

	notice.Reset()
	renderSearchRows([]string{"REF", "TEXT"}, rows, false)
	if notice.Len() != 0 {
		t.Errorf("expected no notice without --full, got %q", notice.String())
	}
}
