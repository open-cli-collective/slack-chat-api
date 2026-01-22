package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

// Helper to create a test client with a mock server
func newTestClient(handler http.HandlerFunc) (*client.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := client.NewWithConfig(server.URL, "xoxp-test-token", nil)
	return c, server
}

func TestRunSearchMessages_Success(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 2,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 2,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C123",
						"name": "general",
					},
					"user":      "U123",
					"username":  "alice",
					"text":      "Test message about deployment",
					"ts":        "1704067200.000000",
					"permalink": "https://slack.com/archives/C123/p1704067200000000",
				},
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C456",
						"name": "engineering",
					},
					"user":      "U456",
					"username":  "bob",
					"text":      "Another deployment message",
					"ts":        "1704067300.000000",
					"permalink": "https://slack.com/archives/C456/p1704067300000000",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "deployment" {
			t.Errorf("unexpected query: %s", r.URL.Query().Get("query"))
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("deployment", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchMessages_NoResults(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 0,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 0,
				"page":  1,
				"pages": 0,
			},
			"matches": []map[string]interface{}{},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("nonexistent", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchMessages_APIError(t *testing.T) {
	response := map[string]interface{}{
		"ok":    false,
		"error": "not_authed",
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("test", opts, c)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRunSearchMessages_InvalidCount(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call with invalid options")
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   101, // Invalid: > 100
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("test", opts, c)
	if err == nil {
		t.Error("expected error for invalid count")
	}
}

func TestRunSearchMessages_InvalidPage(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call with invalid options")
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    101, // Invalid: > 100
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("test", opts, c)
	if err == nil {
		t.Error("expected error for invalid page")
	}
}

func TestRunSearchMessages_InvalidSort(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call with invalid options")
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "invalid",
		sortDir: "desc",
	}

	err := runSearchMessages("test", opts, c)
	if err == nil {
		t.Error("expected error for invalid sort")
	}
}

func TestRunSearchMessages_InvalidSortDir(t *testing.T) {
	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call with invalid options")
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "invalid",
	}

	err := runSearchMessages("test", opts, c)
	if err == nil {
		t.Error("expected error for invalid sort-dir")
	}
}

func TestRunSearchFiles_Success(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"files": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"id":        "F123",
					"name":      "report.pdf",
					"title":     "Quarterly Report",
					"filetype":  "pdf",
					"user":      "U123",
					"created":   1704067200,
					"permalink": "https://slack.com/files/U123/F123/report.pdf",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.files" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &filesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchFiles("report", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchFiles_NoResults(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"files": map[string]interface{}{
			"total": 0,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 0,
				"page":  1,
				"pages": 0,
			},
			"matches": []map[string]interface{}{},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &filesOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchFiles("nonexistent", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchAll_Success(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C123",
						"name": "general",
					},
					"user":      "U123",
					"username":  "alice",
					"text":      "Project proposal discussion",
					"ts":        "1704067200.000000",
					"permalink": "https://slack.com/archives/C123/p1704067200000000",
				},
			},
		},
		"files": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"id":        "F123",
					"name":      "proposal.pdf",
					"title":     "Project Proposal",
					"filetype":  "pdf",
					"user":      "U456",
					"created":   1704067200,
					"permalink": "https://slack.com/files/U456/F123/proposal.pdf",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.all" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &allOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchAll("proposal", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchAll_NoResults(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total":   0,
			"paging":  map[string]interface{}{"count": 20, "total": 0, "page": 1, "pages": 0},
			"matches": []map[string]interface{}{},
		},
		"files": map[string]interface{}{
			"total":   0,
			"paging":  map[string]interface{}{"count": 20, "total": 0, "page": 1, "pages": 0},
			"matches": []map[string]interface{}{},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &allOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchAll("nonexistent", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchAll_OnlyMessages(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C123",
						"name": "general",
					},
					"user":      "U123",
					"username":  "alice",
					"text":      "Test message",
					"ts":        "1704067200.000000",
					"permalink": "https://slack.com/archives/C123/p1704067200000000",
				},
			},
		},
		"files": map[string]interface{}{
			"total":   0,
			"paging":  map[string]interface{}{"count": 20, "total": 0, "page": 1, "pages": 0},
			"matches": []map[string]interface{}{},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &allOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchAll("test", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchAll_OnlyFiles(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total":   0,
			"paging":  map[string]interface{}{"count": 20, "total": 0, "page": 1, "pages": 0},
			"matches": []map[string]interface{}{},
		},
		"files": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"id":        "F123",
					"name":      "test.pdf",
					"title":     "Test File",
					"filetype":  "pdf",
					"user":      "U123",
					"created":   1704067200,
					"permalink": "https://slack.com/files/U123/F123/test.pdf",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &allOptions{
		count:   20,
		page:    1,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchAll("test", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSearchOptions(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		page      int
		sort      string
		sortDir   string
		wantError bool
	}{
		{"valid defaults", 20, 1, "score", "desc", false},
		{"valid timestamp sort", 50, 5, "timestamp", "asc", false},
		{"count too low", 0, 1, "score", "desc", true},
		{"count too high", 101, 1, "score", "desc", true},
		{"page too low", 20, 0, "score", "desc", true},
		{"page too high", 20, 101, "score", "desc", true},
		{"invalid sort", 20, 1, "invalid", "desc", true},
		{"invalid sort-dir", 20, 1, "score", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSearchOptions(tt.count, tt.page, tt.sort, tt.sortDir)
			if (err != nil) != tt.wantError {
				t.Errorf("validateSearchOptions() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short text", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"with newlines", "hello\nworld", 20, "hello world"},
		{"with carriage return", "hello\r\nworld", 20, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateText(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid timestamp", "1704067200.000000", "2024-01-01 00:00"},
		{"no microseconds", "1704067200", "2024-01-01 00:00"},
		{"invalid format", "invalid", "invalid"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimestamp(tt.input)
			// Note: The expected time will vary based on local timezone
			// This test uses UTC expectations; adjust if needed
			if tt.input == "invalid" || tt.input == "" {
				if got != tt.want {
					t.Errorf("formatTimestamp() = %q, want %q", got, tt.want)
				}
			}
			// For valid timestamps, just ensure we get a non-empty result
			if tt.input != "invalid" && tt.input != "" && got == "" {
				t.Errorf("formatTimestamp() returned empty for valid input")
			}
		})
	}
}

func TestFormatUnixTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"valid timestamp", 1704067200, "2024-01-01"},
		{"zero", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUnixTimestamp(tt.input)
			if tt.input == 0 {
				if got != tt.want {
					t.Errorf("formatUnixTimestamp() = %q, want %q", got, tt.want)
				}
			} else {
				// For non-zero, just ensure we get a date format
				if len(got) != 10 { // YYYY-MM-DD format
					t.Errorf("formatUnixTimestamp() = %q, expected date format", got)
				}
			}
		})
	}
}

func TestSearchMessages_WithPagination(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 100,
			"paging": map[string]interface{}{
				"count": 10,
				"total": 100,
				"page":  3,
				"pages": 10,
			},
			"matches": []map[string]interface{}{
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C123",
						"name": "general",
					},
					"user":      "U123",
					"username":  "alice",
					"text":      "Test message",
					"ts":        "1704067200.000000",
					"permalink": "https://slack.com/archives/C123/p1704067200000000",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		// Verify pagination params are passed
		if r.URL.Query().Get("count") != "10" {
			t.Errorf("expected count=10, got %s", r.URL.Query().Get("count"))
		}
		if r.URL.Query().Get("page") != "3" {
			t.Errorf("expected page=3, got %s", r.URL.Query().Get("page"))
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   10,
		page:    3,
		sort:    "score",
		sortDir: "desc",
	}

	err := runSearchMessages("test", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchMessages_WithHighlight(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total": 1,
			"paging": map[string]interface{}{
				"count": 20,
				"total": 1,
				"page":  1,
				"pages": 1,
			},
			"matches": []map[string]interface{}{
				{
					"type": "message",
					"channel": map[string]interface{}{
						"id":   "C123",
						"name": "general",
					},
					"user":      "U123",
					"username":  "alice",
					"text":      "Test <mark>deployment</mark> message",
					"ts":        "1704067200.000000",
					"permalink": "https://slack.com/archives/C123/p1704067200000000",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("highlight") != "true" {
			t.Errorf("expected highlight=true, got %s", r.URL.Query().Get("highlight"))
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:     20,
		page:      1,
		sort:      "score",
		sortDir:   "desc",
		highlight: true,
	}

	err := runSearchMessages("deployment", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSearchMessages_WithSortOptions(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"total":   0,
			"paging":  map[string]interface{}{"count": 20, "total": 0, "page": 1, "pages": 0},
			"matches": []map[string]interface{}{},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sort") != "timestamp" {
			t.Errorf("expected sort=timestamp, got %s", r.URL.Query().Get("sort"))
		}
		if r.URL.Query().Get("sort_dir") != "asc" {
			t.Errorf("expected sort_dir=asc, got %s", r.URL.Query().Get("sort_dir"))
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &messagesOptions{
		count:   20,
		page:    1,
		sort:    "timestamp",
		sortDir: "asc",
	}

	err := runSearchMessages("test", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
