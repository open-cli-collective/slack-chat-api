package messages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		expectSame bool // If true, expect output equals input (for invalid inputs)
	}{
		{
			name:  "standard timestamp",
			input: "1704067200.123456",
		},
		{
			name:  "timestamp without decimal",
			input: "1704067200",
		},
		{
			name:       "empty string",
			input:      "",
			expectSame: true,
		},
		{
			name:       "invalid timestamp",
			input:      "not-a-timestamp",
			expectSame: true,
		},
		{
			name:  "timestamp with extra precision",
			input: "1704067200.123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimestamp(tt.input)
			if tt.expectSame {
				if result != tt.input {
					t.Errorf("formatTimestamp(%q) = %q, expected %q", tt.input, result, tt.input)
				}
			} else {
				// For valid timestamps, check the format is correct (YYYY-MM-DD HH:MM)
				if len(result) != 16 {
					t.Errorf("formatTimestamp(%q) = %q, expected 16-char format YYYY-MM-DD HH:MM", tt.input, result)
				}
				// Check it contains expected delimiters
				if result[4] != '-' || result[7] != '-' || result[10] != ' ' || result[13] != ':' {
					t.Errorf("formatTimestamp(%q) = %q, format doesn't match YYYY-MM-DD HH:MM", tt.input, result)
				}
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no newlines",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "single newline",
			input:    "Hello\nWorld",
			expected: "Hello World",
		},
		{
			name:     "multiple newlines",
			input:    "Line1\nLine2\nLine3",
			expected: "Line1 Line2 Line3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "long string preserved",
			input:    "This is a very long message that exceeds eighty characters and should not be truncated at all by the flatten function",
			expected: "This is a very long message that exceeds eighty characters and should not be truncated at all by the flatten function",
		},
		{
			name:     "long string with newlines preserved",
			input:    "First line\nSecond line that is quite long\nThird line with more content that pushes past eighty characters total",
			expected: "First line Second line that is quite long Third line with more content that pushes past eighty characters total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flatten(tt.input)
			if result != tt.expected {
				t.Errorf("flatten(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string no truncation",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "exact length",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "truncation needed",
			input:    "Hello World!",
			maxLen:   8,
			expected: "Hello...",
		},
		{
			name:     "newlines converted to spaces",
			input:    "Hello\nWorld",
			maxLen:   20,
			expected: "Hello World",
		},
		{
			name:     "multiple newlines",
			input:    "Line1\nLine2\nLine3",
			maxLen:   20,
			expected: "Line1 Line2 Line3",
		},
		{
			name:     "truncation with newlines",
			input:    "Hello\nWorld\nFoo\nBar",
			maxLen:   10,
			expected: "Hello W...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestUnescapeShellChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escape sequences",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "escaped exclamation mark",
			input:    `Hello\! World\!`,
			expected: "Hello! World!",
		},
		{
			name:     "multiple escaped exclamation marks",
			input:    `Test\!\!\!`,
			expected: "Test!!!",
		},
		{
			name:     "mixed content",
			input:    `Hello\! This is a *bold* message\!`,
			expected: "Hello! This is a *bold* message!",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only backslash (not escaping !)",
			input:    `Hello\nWorld`,
			expected: `Hello\nWorld`,
		},
		{
			name:     "backslash at end",
			input:    `Hello\`,
			expected: `Hello\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unescapeShellChars(tt.input)
			if result != tt.expected {
				t.Errorf("unescapeShellChars(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildDefaultBlocks(t *testing.T) {
	t.Run("simple text", func(t *testing.T) {
		result := buildDefaultBlocks("Hello World")
		assert.Len(t, result, 1)
		block := result[0].(map[string]interface{})
		assert.Equal(t, "section", block["type"])
		textObj := block["text"].(map[string]interface{})
		assert.Equal(t, "mrkdwn", textObj["type"])
		assert.Equal(t, "Hello World", textObj["text"])
	})

	t.Run("empty text", func(t *testing.T) {
		result := buildDefaultBlocks("")
		assert.Len(t, result, 1)
		block := result[0].(map[string]interface{})
		textObj := block["text"].(map[string]interface{})
		assert.Equal(t, "", textObj["text"])
	})

	t.Run("text exceeding 3000 chars splits into multiple blocks", func(t *testing.T) {
		// Build text with lines that total > 3000 chars
		var lines []string
		for i := 0; i < 100; i++ {
			lines = append(lines, strings.Repeat("x", 50))
		}
		text := strings.Join(lines, "\n") // 100 lines * 51 chars = 5100 chars

		result := buildDefaultBlocks(text)
		assert.Greater(t, len(result), 1, "should split into multiple blocks")

		// Verify all blocks are valid section blocks with text under the limit
		var reconstructed string
		for i, b := range result {
			block := b.(map[string]interface{})
			assert.Equal(t, "section", block["type"])
			textObj := block["text"].(map[string]interface{})
			assert.Equal(t, "mrkdwn", textObj["type"])
			chunk := textObj["text"].(string)
			assert.LessOrEqual(t, len(chunk), maxSectionTextLen,
				"block %d exceeds max section text length", i)
			if i > 0 {
				reconstructed += "\n"
			}
			reconstructed += chunk
		}
		assert.Equal(t, text, reconstructed, "reconstructed text should match original")
	})

	t.Run("text at exactly 3000 chars stays as one block", func(t *testing.T) {
		text := strings.Repeat("a", 3000)
		result := buildDefaultBlocks(text)
		assert.Len(t, result, 1)
	})

	t.Run("text at 3001 chars splits", func(t *testing.T) {
		text := strings.Repeat("a", 3001)
		result := buildDefaultBlocks(text)
		assert.Greater(t, len(result), 1)
	})
}

// Command handler tests

func TestRunSend_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat.postMessage", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "Hello World", body["text"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C123",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{simple: true}

	err := runSend("C123", "Hello World", opts, c)
	require.NoError(t, err)
}

func TestRunSend_WithThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "1234567890.000000", body["thread_ts"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{threadTS: "1234567890.000000", simple: true}

	err := runSend("C123", "Reply", opts, c)
	require.NoError(t, err)
}

func TestRunSend_WithBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		blocks := body["blocks"].([]interface{})
		assert.Len(t, blocks, 1)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{blocksJSON: `[{"type":"section","text":{"type":"mrkdwn","text":"Hello"}}]`}

	err := runSend("C123", "Hello", opts, c)
	require.NoError(t, err)
}

func TestRunSend_InvalidBlocks(t *testing.T) {
	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &sendOptions{blocksJSON: "not valid json"}

	err := runSend("C123", "Hello", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blocks JSON")
}

func TestRunUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat.update", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "1234567890.123456", body["ts"])
		assert.Equal(t, "Updated text", body["text"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &updateOptions{simple: true}

	err := runUpdate("C123", "1234567890.123456", "Updated text", opts, c)
	require.NoError(t, err)
}

func TestRunDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat.delete", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "1234567890.123456", body["ts"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &deleteOptions{}

	err := runDelete("C123", "1234567890.123456", opts, c)
	require.NoError(t, err)
}

func mockUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user")
	names := map[string]string{
		"U001": "alice",
		"U002": "bob",
	}
	name := names[userID]
	if name == "" {
		name = userID
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true,
		"user": map[string]interface{}{
			"id":        userID,
			"name":      name,
			"real_name": name,
			"profile": map[string]interface{}{
				"display_name": name,
			},
		},
	})
}

func TestRunHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.history":
			assert.Equal(t, "C123", r.URL.Query().Get("channel"))
			assert.Equal(t, "20", r.URL.Query().Get("limit"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1234567890.123456", "user": "U001", "text": "Hello"},
					{"ts": "1234567890.123457", "user": "U002", "text": "World"},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	err := runHistory("C123", opts, c)
	require.NoError(t, err)
}

func TestRunHistory_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.history":
			assert.Equal(t, "1234567890.000000", r.URL.Query().Get("oldest"))
			assert.Equal(t, "1234567899.000000", r.URL.Query().Get("latest"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":       true,
				"messages": []map[string]interface{}{},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{
		limit:  20,
		oldest: "1234567890.000000",
		latest: "1234567899.000000",
	}

	err := runHistory("C123", opts, c)
	require.NoError(t, err)
}

func TestRunHistory_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":       true,
				"messages": []map[string]interface{}{},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	err := runHistory("C123", opts, c)
	require.NoError(t, err)
}

func TestRunThread_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.replies":
			assert.Equal(t, "C123", r.URL.Query().Get("channel"))
			assert.Equal(t, "1234567890.123456", r.URL.Query().Get("ts"))
			assert.Equal(t, "100", r.URL.Query().Get("limit"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{"ts": "1234567890.123456", "user": "U001", "text": "Original"},
					{"ts": "1234567890.123457", "user": "U002", "text": "Reply 1"},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &threadOptions{limit: 100}

	err := runThread("C123", "1234567890.123456", opts, c)
	require.NoError(t, err)
}

func TestRunThread_JSONIncludesReactions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.replies":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts":   "1234567890.123456",
						"user": "U001",
						"text": "Original",
						"reactions": []map[string]interface{}{
							{"name": "thumbsup", "count": 2, "users": []string{"U001", "U002"}},
						},
					},
					{
						"ts":   "1234567890.123457",
						"user": "U002",
						"text": "Reply without reactions",
					},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &threadOptions{limit: 100}

	// Capture JSON output
	output.OutputFormat = output.FormatJSON
	defer func() { output.OutputFormat = output.FormatText }()

	var buf strings.Builder
	output.Writer = &buf
	defer func() { output.Writer = os.Stdout }()

	err := runThread("C123", "1234567890.123456", opts, c)
	require.NoError(t, err)

	// Parse the JSON output
	var messages []client.Message
	err = json.Unmarshal([]byte(buf.String()), &messages)
	require.NoError(t, err)

	require.Len(t, messages, 2)

	// First message should have reactions
	require.Len(t, messages[0].Reactions, 1)
	assert.Equal(t, "thumbsup", messages[0].Reactions[0].Name)
	assert.Equal(t, 2, messages[0].Reactions[0].Count)
	assert.Equal(t, []string{"U001", "U002"}, messages[0].Reactions[0].Users)

	// Second message should have no reactions
	assert.Empty(t, messages[1].Reactions)
}

func TestRunThread_JSONIncludesFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.replies":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts":   "1234567890.123456",
						"user": "U001",
						"text": "See the screenshot below",
						"files": []map[string]interface{}{
							{
								"id":                   "F0123ABC",
								"name":                 "screenshot.png",
								"title":                "Screenshot",
								"mimetype":             "image/png",
								"filetype":             "png",
								"size":                 54321,
								"url_private":          "https://files.slack.com/files-pri/T123-F0123ABC/screenshot.png",
								"url_private_download": "https://files.slack.com/files-pri/T123-F0123ABC/download/screenshot.png",
								"permalink":            "https://example.slack.com/files/U001/F0123ABC/screenshot.png",
							},
						},
					},
					{
						"ts":   "1234567890.123457",
						"user": "U002",
						"text": "Plain reply without files",
					},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &threadOptions{limit: 100}

	// Capture JSON output
	output.OutputFormat = output.FormatJSON
	defer func() { output.OutputFormat = output.FormatText }()

	var buf strings.Builder
	output.Writer = &buf
	defer func() { output.Writer = os.Stdout }()

	err := runThread("C123", "1234567890.123456", opts, c)
	require.NoError(t, err)

	// Parse the JSON output
	var messages []client.Message
	err = json.Unmarshal([]byte(buf.String()), &messages)
	require.NoError(t, err)

	require.Len(t, messages, 2)

	// First message should have files
	require.Len(t, messages[0].Files, 1)
	assert.Equal(t, "F0123ABC", messages[0].Files[0].ID)
	assert.Equal(t, "screenshot.png", messages[0].Files[0].Name)
	assert.Equal(t, "image/png", messages[0].Files[0].Mimetype)
	assert.Equal(t, "png", messages[0].Files[0].Filetype)
	assert.Equal(t, int64(54321), messages[0].Files[0].Size)
	assert.Equal(t, "https://files.slack.com/files-pri/T123-F0123ABC/screenshot.png", messages[0].Files[0].URLPrivate)
	assert.Equal(t, "https://example.slack.com/files/U001/F0123ABC/screenshot.png", messages[0].Files[0].Permalink)

	// Second message should have no files
	assert.Empty(t, messages[1].Files)
}

func TestRunHistory_JSONIncludesReactions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts":   "1234567890.123456",
						"user": "U001",
						"text": "Hello",
						"reactions": []map[string]interface{}{
							{"name": "wave", "count": 1, "users": []string{"U002"}},
							{"name": "heart", "count": 3, "users": []string{"U001", "U002", "U003"}},
						},
					},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	// Capture JSON output
	output.OutputFormat = output.FormatJSON
	defer func() { output.OutputFormat = output.FormatText }()

	var buf strings.Builder
	output.Writer = &buf
	defer func() { output.Writer = os.Stdout }()

	err := runHistory("C123", opts, c)
	require.NoError(t, err)

	// Parse the JSON output
	var messages []client.Message
	err = json.Unmarshal([]byte(buf.String()), &messages)
	require.NoError(t, err)

	require.Len(t, messages, 1)
	require.Len(t, messages[0].Reactions, 2)
	assert.Equal(t, "wave", messages[0].Reactions[0].Name)
	assert.Equal(t, "heart", messages[0].Reactions[1].Name)
	assert.Equal(t, 3, messages[0].Reactions[1].Count)
}

func TestRunReact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/reactions.add", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "1234567890.123456", body["timestamp"])
		assert.Equal(t, "thumbsup", body["name"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &reactOptions{}

	err := runReact("C123", "1234567890.123456", "thumbsup", opts, c)
	require.NoError(t, err)
}

func TestRunReact_StripsColons(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "thumbsup", body["name"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &reactOptions{}

	err := runReact("C123", "1234567890.123456", ":thumbsup:", opts, c)
	require.NoError(t, err)
}

func TestRunUnreact_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/reactions.remove", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "1234567890.123456", body["timestamp"])
		assert.Equal(t, "thumbsup", body["name"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &unreactOptions{}

	err := runUnreact("C123", "1234567890.123456", ":thumbsup:", opts, c)
	require.NoError(t, err)
}

// Confirmation prompt tests for delete command

func TestRunDelete_Confirmation(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		force         bool
		expectAPICall bool
	}{
		{"force skips prompt", "", true, true},
		{"y confirms", "y\n", false, true},
		{"yes confirms", "yes\n", false, true},
		{"YES confirms (case insensitive)", "YES\n", false, true},
		{"n cancels", "n\n", false, false},
		{"no cancels", "no\n", false, false},
		{"empty input cancels", "\n", false, false},
		{"other input cancels", "maybe\n", false, false},
		{"whitespace y confirms", "  y  \n", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalled := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				apiCalled = true
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
			}))
			defer server.Close()

			c := client.NewWithConfig(server.URL, "test-token", nil)
			opts := &deleteOptions{
				force: tt.force,
				stdin: strings.NewReader(tt.input),
			}

			err := runDelete("C123456789", "1234567890.123456", opts, c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectAPICall, apiCalled, "API call expectation mismatch")
		})
	}
}

// Stdin support tests for send command

func TestRunSend_Stdin(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectText  string
		expectError bool
	}{
		{
			name:       "single line from stdin",
			input:      "Hello from stdin",
			expectText: "Hello from stdin",
		},
		{
			name:       "multiline preserves newlines",
			input:      "Line 1\nLine 2\nLine 3",
			expectText: "Line 1\nLine 2\nLine 3",
		},
		{
			name:       "unicode and emoji preserved",
			input:      "Hello 👋 World 🌍",
			expectText: "Hello 👋 World 🌍",
		},
		{
			name:        "empty stdin fails",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedText string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				receivedText = body["text"].(string)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"ok": true,
					"ts": "1234567890.123456",
				})
			}))
			defer server.Close()

			c := client.NewWithConfig(server.URL, "test-token", nil)
			opts := &sendOptions{
				simple: true,
				stdin:  strings.NewReader(tt.input),
			}

			err := runSend("C123456789", "-", opts, c)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "empty")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectText, receivedText)
			}
		})
	}
}

// Validation tests

func TestRunSend_UnescapesShellChars(t *testing.T) {
	var receivedText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedText = body["text"].(string)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{simple: true}

	// Simulate what zsh does: escapes ! as \!
	err := runSend("C123456789", `Hello\! Thanks\!`, opts, c)
	require.NoError(t, err)
	// The CLI should unescape \! back to !
	assert.Equal(t, "Hello! Thanks!", receivedText)
}

func TestRunSend_UnescapesStdinContent(t *testing.T) {
	var receivedText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedText = body["text"].(string)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{
		simple: true,
		stdin:  strings.NewReader(`Hello\! From stdin\!`),
	}

	err := runSend("C123456789", "-", opts, c)
	require.NoError(t, err)
	// Stdin content should also be unescaped
	assert.Equal(t, "Hello! From stdin!", receivedText)
}

func TestRunUpdate_UnescapesShellChars(t *testing.T) {
	var receivedText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedText = body["text"].(string)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &updateOptions{simple: true}

	err := runUpdate("C123456789", "1234567890.123456", `Updated\! Text\!`, opts, c)
	require.NoError(t, err)
	assert.Equal(t, "Updated! Text!", receivedText)
}

func TestRunSend_ChannelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{simple: true}
	err := runSend("nonexistent-channel", "Hello", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunSend_InvalidThreadTimestamp(t *testing.T) {
	opts := &sendOptions{simple: true, threadTS: "not-a-timestamp"}
	err := runSend("C123456789", "Hello", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunDelete_ChannelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &deleteOptions{force: true}
	err := runDelete("nonexistent-channel", "1234567890.123456", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunDelete_InvalidTimestamp(t *testing.T) {
	opts := &deleteOptions{force: true}
	err := runDelete("C123456789", "not-a-timestamp", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunReact_ChannelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &reactOptions{}
	err := runReact("nonexistent-channel", "1234567890.123456", "thumbsup", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunReact_InvalidTimestamp(t *testing.T) {
	opts := &reactOptions{}
	err := runReact("C123456789", "not-a-timestamp", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunUnreact_ChannelNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &unreactOptions{}
	err := runUnreact("nonexistent-channel", "1234567890.123456", "thumbsup", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunUnreact_InvalidTimestamp(t *testing.T) {
	opts := &unreactOptions{}
	err := runUnreact("C123456789", "not-a-timestamp", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunReact_AlreadyReacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "already_reacted",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &reactOptions{}

	err := runReact("C123", "1234567890.123456", "thumbsup", opts, c)
	require.NoError(t, err)
}

func TestRunUnreact_NoReaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "no_reaction",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &unreactOptions{}

	err := runUnreact("C123", "1234567890.123456", "thumbsup", opts, c)
	require.NoError(t, err)
}

// Tests for blocks-file and blocks-stdin features

func TestRunSend_BlocksOnly(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{blocksJSON: `[{"type":"section","text":{"type":"mrkdwn","text":"Hello from blocks"}}]`}

	// Empty text, blocks only
	err := runSend("C123456789", "", opts, c)
	require.NoError(t, err)

	// Verify text was not sent (Slack allows blocks without text)
	_, hasText := receivedBody["text"]
	assert.False(t, hasText, "text should not be included when empty")

	// Verify blocks were sent
	blocks := receivedBody["blocks"].([]interface{})
	assert.Len(t, blocks, 1)
}

func TestRunSend_BlocksFile(t *testing.T) {
	// Create a temporary blocks file
	tmpFile, err := os.CreateTemp("", "blocks-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	blocksJSON := `[{"type":"section","text":{"type":"mrkdwn","text":"From file"}}]`
	_, err = tmpFile.WriteString(blocksJSON)
	require.NoError(t, err)
	tmpFile.Close()

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{blocksFile: tmpFile.Name()}

	err = runSend("C123456789", "Fallback text", opts, c)
	require.NoError(t, err)

	// Verify blocks were parsed from file
	blocks := receivedBody["blocks"].([]interface{})
	assert.Len(t, blocks, 1)
	section := blocks[0].(map[string]interface{})
	textObj := section["text"].(map[string]interface{})
	assert.Equal(t, "From file", textObj["text"])
}

func TestRunSend_BlocksFileOnly(t *testing.T) {
	// Create a temporary blocks file
	tmpFile, err := os.CreateTemp("", "blocks-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	blocksJSON := `[{"type":"section","text":{"type":"mrkdwn","text":"From file only"}}]`
	_, err = tmpFile.WriteString(blocksJSON)
	require.NoError(t, err)
	tmpFile.Close()

	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{blocksFile: tmpFile.Name()}

	// No text, only blocks from file
	err = runSend("C123456789", "", opts, c)
	require.NoError(t, err)

	// Verify text was not sent
	_, hasText := receivedBody["text"]
	assert.False(t, hasText, "text should not be included when empty")

	// Verify blocks were sent
	blocks := receivedBody["blocks"].([]interface{})
	assert.Len(t, blocks, 1)
}

func TestRunSend_BlocksFileNotFound(t *testing.T) {
	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &sendOptions{blocksFile: "/nonexistent/file.json"}

	err := runSend("C123456789", "text", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading blocks file")
}

func TestRunSend_BlocksFileInvalidJSON(t *testing.T) {
	// Create a temporary blocks file with invalid JSON
	tmpFile, err := os.CreateTemp("", "blocks-*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("not valid json")
	require.NoError(t, err)
	tmpFile.Close()

	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &sendOptions{blocksFile: tmpFile.Name()}

	err = runSend("C123456789", "text", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blocks JSON")
}

func TestRunSend_BlocksStdin(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	blocksJSON := `[{"type":"section","text":{"type":"mrkdwn","text":"From stdin"}}]`
	opts := &sendOptions{
		blocksStdin: true,
		stdin:       strings.NewReader(blocksJSON),
	}

	err := runSend("C123456789", "Fallback text", opts, c)
	require.NoError(t, err)

	// Verify blocks were parsed from stdin
	blocks := receivedBody["blocks"].([]interface{})
	assert.Len(t, blocks, 1)
	section := blocks[0].(map[string]interface{})
	textObj := section["text"].(map[string]interface{})
	assert.Equal(t, "From stdin", textObj["text"])
}

func TestRunSend_BlocksStdinOnly(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	blocksJSON := `[{"type":"section","text":{"type":"mrkdwn","text":"Stdin only"}}]`
	opts := &sendOptions{
		blocksStdin: true,
		stdin:       strings.NewReader(blocksJSON),
	}

	// No text, only blocks from stdin
	err := runSend("C123456789", "", opts, c)
	require.NoError(t, err)

	// Verify text was not sent
	_, hasText := receivedBody["text"]
	assert.False(t, hasText, "text should not be included when empty")

	// Verify blocks were sent
	blocks := receivedBody["blocks"].([]interface{})
	assert.Len(t, blocks, 1)
}

func TestRunSend_BlocksStdinInvalidJSON(t *testing.T) {
	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &sendOptions{
		blocksStdin: true,
		stdin:       strings.NewReader("not valid json"),
	}

	err := runSend("C123456789", "text", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid blocks JSON")
}

func TestRunSend_MutuallyExclusiveBlocksOptions(t *testing.T) {
	tests := []struct {
		name string
		opts *sendOptions
	}{
		{
			name: "blocks and blocks-file",
			opts: &sendOptions{
				blocksJSON: `[{"type":"section"}]`,
				blocksFile: "/some/file.json",
			},
		},
		{
			name: "blocks and blocks-stdin",
			opts: &sendOptions{
				blocksJSON:  `[{"type":"section"}]`,
				blocksStdin: true,
			},
		},
		{
			name: "blocks-file and blocks-stdin",
			opts: &sendOptions{
				blocksFile:  "/some/file.json",
				blocksStdin: true,
			},
		},
		{
			name: "all three options",
			opts: &sendOptions{
				blocksJSON:  `[{"type":"section"}]`,
				blocksFile:  "/some/file.json",
				blocksStdin: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runSend("C123456789", "text", tt.opts, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "only one of --blocks, --blocks-file, or --blocks-stdin")
		})
	}
}

func TestRunSend_TextStdinAndBlocksStdinConflict(t *testing.T) {
	opts := &sendOptions{
		blocksStdin: true,
		stdin:       strings.NewReader("some content"),
	}

	// Using "-" for text means reading text from stdin, which conflicts with --blocks-stdin
	err := runSend("C123456789", "-", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use '-' for text and --blocks-stdin together")
}

func TestRunSend_EmptyTextNoBlocks(t *testing.T) {
	opts := &sendOptions{}

	err := runSend("C123456789", "", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message text cannot be empty")
}

func TestRunSend_NoUnfurl(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{simple: true, noUnfurl: true}

	err := runSend("C123456789", "Check https://example.com", opts, c)
	require.NoError(t, err)

	// Verify unfurl parameters are set to false
	assert.Equal(t, false, receivedBody["unfurl_links"])
	assert.Equal(t, false, receivedBody["unfurl_media"])
}

func TestRunSend_UnfurlEnabled(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{simple: true, noUnfurl: false}

	err := runSend("C123456789", "Check https://example.com", opts, c)
	require.NoError(t, err)

	// Verify unfurl parameters are set to true (default behavior)
	assert.Equal(t, true, receivedBody["unfurl_links"])
	assert.Equal(t, true, receivedBody["unfurl_media"])
}

func TestRunUpdate_NoUnfurl(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &updateOptions{simple: true, noUnfurl: true}

	err := runUpdate("C123456789", "1234567890.123456", "Updated https://example.com", opts, c)
	require.NoError(t, err)

	// Verify unfurl parameters are set to false
	assert.Equal(t, false, receivedBody["unfurl_links"])
	assert.Equal(t, false, receivedBody["unfurl_media"])
}

func TestRunUpdate_UnfurlEnabled(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &updateOptions{simple: true, noUnfurl: false}

	err := runUpdate("C123456789", "1234567890.123456", "Updated https://example.com", opts, c)
	require.NoError(t, err)

	// Verify unfurl parameters are set to true (default behavior)
	assert.Equal(t, true, receivedBody["unfurl_links"])
	assert.Equal(t, true, receivedBody["unfurl_media"])
}

func TestRunSend_FileUpload(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-upload-*.txt")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("hello world")
	require.NoError(t, err)
	tmpFile.Close()

	step := 0

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "files.getUploadURLExternal"):
			step++
			resp := map[string]interface{}{
				"ok":         true,
				"upload_url": uploadServer.URL + "/upload",
				"file_id":    "F123",
			}
			json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "files.completeUploadExternal"):
			step++
			resp := map[string]interface{}{"ok": true}
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{
		files: []string{tmpFile.Name()},
	}

	err = runSend("C123456789", "", opts, c)
	require.NoError(t, err)
	assert.Equal(t, 2, step, "should complete both API calls")
}

func TestRunSend_FileUpload_WithComment(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-upload-*.txt")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("content")
	require.NoError(t, err)
	tmpFile.Close()

	var completeBody map[string]interface{}

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "files.getUploadURLExternal"):
			resp := map[string]interface{}{
				"ok":         true,
				"upload_url": uploadServer.URL + "/upload",
				"file_id":    "F456",
			}
			json.NewEncoder(w).Encode(resp)
		case strings.Contains(r.URL.Path, "files.completeUploadExternal"):
			json.NewDecoder(r.Body).Decode(&completeBody)
			resp := map[string]interface{}{"ok": true}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{
		files:     []string{tmpFile.Name()},
		fileTitle: "My Report",
	}

	err = runSend("C123456789", "Here's the report", opts, c)
	require.NoError(t, err)

	assert.Equal(t, "C123456789", completeBody["channel_id"])
	assert.Equal(t, "Here's the report", completeBody["initial_comment"])
}

func TestSendCmd_ChannelFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "Hello", body["text"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{channel: "C123", simple: true}

	err := runSend("C123", "Hello", opts, c)
	require.NoError(t, err)
}

func TestSendCmd_ChannelFlagWithMultipleArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "Hello World", body["text"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"ts": "1234567890.123456",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	// Simulate what RunE does: join args when --channel is set
	opts := &sendOptions{channel: "C123", simple: true}
	text := strings.Join([]string{"Hello", "World"}, " ")

	err := runSend("C123", text, opts, c)
	require.NoError(t, err)
}

func TestSendCmd_NoChannelError(t *testing.T) {
	cmd := newSendCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
}

func TestRunSend_FileUpload_NonexistentFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API calls for nonexistent file")
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &sendOptions{
		files: []string{"/nonexistent/file.txt"},
	}

	err := runSend("C123456789", "", opts, c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access file")
}

func TestValidateMessageLength(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{"under limit", strings.Repeat("x", 1000), false},
		{"at limit", strings.Repeat("x", maxMessageTextLen), false},
		{"over limit by one", strings.Repeat("x", maxMessageTextLen+1), true},
		{"well over limit", strings.Repeat("x", 50000), true},
		{"empty", "", false},
		{"multi-byte under limit", strings.Repeat("😀", 1000), false},
		{"multi-byte at limit", strings.Repeat("😀", maxMessageTextLen), false},
		{"multi-byte over limit", strings.Repeat("😀", maxMessageTextLen+1), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMessageLength(tt.text, false)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds Slack's 40000 character limit")
				assert.Contains(t, err.Error(), "--file")
				assert.Contains(t, err.Error(), "canvas create")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateMessageLength_UpdateOmitsFileHint(t *testing.T) {
	text := strings.Repeat("x", maxMessageTextLen+1)
	err := validateMessageLength(text, true)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "--file")
	assert.Contains(t, err.Error(), "canvas create")
}

func TestRunSend_MessageTooLong(t *testing.T) {
	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &sendOptions{simple: true}

	longText := strings.Repeat("x", maxMessageTextLen+1)
	err := runSend("C123", longText, opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds Slack's 40000 character limit")
}

func TestRunUpdate_MessageTooLong(t *testing.T) {
	c := client.NewWithConfig("http://localhost", "test-token", nil)
	opts := &updateOptions{simple: true}

	longText := strings.Repeat("x", maxMessageTextLen+1)
	err := runUpdate("C123", "1234567890.123456", longText, opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds Slack's 40000 character limit")
}

func TestRunSend_FileUpload_LongTextAllowed(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test-upload-*.txt")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("content")
	require.NoError(t, err)
	tmpFile.Close()

	uploadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "files.getUploadURLExternal"):
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":         true,
				"upload_url": uploadServer.URL + "/upload",
				"file_id":    "F123",
			})
		case strings.Contains(r.URL.Path, "files.completeUploadExternal"):
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	longText := strings.Repeat("x", maxMessageTextLen+1)
	opts := &sendOptions{
		files: []string{tmpFile.Name()},
	}

	err = runSend("C123456789", longText, opts, c)
	require.NoError(t, err)
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		n        int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{411, "411 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{12345, "12.1 KB"},
		{1024 * 1024, "1.0 MB"},
		{4_718_592, "4.5 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, humanSize(tt.n))
		})
	}
}

func TestRenderFiles(t *testing.T) {
	t.Run("empty returns empty string", func(t *testing.T) {
		assert.Equal(t, "", renderFiles(nil))
		assert.Equal(t, "", renderFiles([]client.File{}))
	})

	t.Run("single file uses name when title is empty", func(t *testing.T) {
		got := renderFiles([]client.File{{
			ID:       "F0ABC",
			Name:     "data.csv",
			Filetype: "csv",
			Size:     411,
		}})
		assert.Equal(t, "\t[file] data.csv (csv, 411 B) — slck files download F0ABC\n", got)
	})

	t.Run("title takes precedence over name", func(t *testing.T) {
		got := renderFiles([]client.File{{
			ID:       "F0ABC",
			Name:     "raw_upload_12345.png",
			Title:    "Screenshot",
			Filetype: "png",
			Size:     54321,
		}})
		assert.Contains(t, got, "Screenshot")
		assert.NotContains(t, got, "raw_upload_12345.png")
	})

	t.Run("multiple files render one line each", func(t *testing.T) {
		got := renderFiles([]client.File{
			{ID: "F1", Name: "a.csv", Filetype: "csv", Size: 100},
			{ID: "F2", Name: "b.pdf", Filetype: "pdf", Size: 2048},
		})
		lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
		require.Len(t, lines, 2)
		assert.Contains(t, lines[0], "F1")
		assert.Contains(t, lines[0], "a.csv")
		assert.Contains(t, lines[1], "F2")
		assert.Contains(t, lines[1], "b.pdf")
	})

	t.Run("each line starts with tab prefix", func(t *testing.T) {
		got := renderFiles([]client.File{{ID: "F1", Name: "a.csv", Filetype: "csv", Size: 100}})
		assert.True(t, strings.HasPrefix(got, "\t"), "expected line to start with tab, got %q", got)
	})
}

// captureTextOutput captures text output for a test function and resets state.
func captureTextOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf strings.Builder
	origWriter := output.Writer
	output.Writer = &buf
	defer func() { output.Writer = origWriter }()
	fn()
	return buf.String()
}

func TestRunThread_TextIncludesFileHints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.replies":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts":   "1234567890.123456",
						"user": "U001",
						"text": "Here are the check files",
						"files": []map[string]interface{}{
							{
								"id":       "F0AT13FGVAT",
								"name":     "data.csv",
								"filetype": "csv",
								"size":     411,
							},
						},
					},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &threadOptions{limit: 100}

	out := captureTextOutput(t, func() {
		require.NoError(t, runThread("C123", "1234567890.123456", opts, c))
	})

	assert.Contains(t, out, "[file]")
	assert.Contains(t, out, "data.csv")
	assert.Contains(t, out, "csv")
	assert.Contains(t, out, "411 B")
	assert.Contains(t, out, "slck files download F0AT13FGVAT")
}

func TestRunHistory_TextIncludesFileHints(t *testing.T) {
	// Message text intentionally long to ensure file hint still renders in full
	longText := "This is a very long message body that definitely exceeds eighty characters and will be truncated in history's compact view."
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/conversations.history":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"messages": []map[string]interface{}{
					{
						"ts":   "1234567890.123456",
						"user": "U001",
						"text": longText,
						"files": []map[string]interface{}{
							{
								"id":       "F0ATD4WJ70D",
								"name":     "IW Trailer.pdf",
								"filetype": "pdf",
								"size":     60646,
							},
						},
					},
				},
			})
		case "/users.info":
			mockUserInfoHandler(w, r)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	out := captureTextOutput(t, func() {
		require.NoError(t, runHistory("C123", opts, c))
	})

	assert.Contains(t, out, "[file]")
	assert.Contains(t, out, "IW Trailer.pdf")
	assert.Contains(t, out, "pdf")
	assert.Contains(t, out, "slck files download F0ATD4WJ70D")
	// File hint must render in full even though the message body is truncated
	assert.NotContains(t, out, "...F0ATD4WJ70D")
}
