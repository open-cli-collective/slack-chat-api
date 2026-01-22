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
	tests := []struct {
		name string
		text string
	}{
		{
			name: "simple text",
			text: "Hello World",
		},
		{
			name: "markdown text",
			text: "*bold* _italic_ ~strike~",
		},
		{
			name: "empty text",
			text: "",
		},
		{
			name: "text with special characters",
			text: "Hello <@U123> in #general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDefaultBlocks(tt.text)

			if len(result) != 1 {
				t.Fatalf("expected 1 block, got %d", len(result))
			}

			block, ok := result[0].(map[string]interface{})
			if !ok {
				t.Fatal("expected block to be map[string]interface{}")
			}

			if block["type"] != "section" {
				t.Errorf("expected block type 'section', got %v", block["type"])
			}

			textObj, ok := block["text"].(map[string]interface{})
			if !ok {
				t.Fatal("expected text to be map[string]interface{}")
			}

			if textObj["type"] != "mrkdwn" {
				t.Errorf("expected text type 'mrkdwn', got %v", textObj["type"])
			}

			if textObj["text"] != tt.text {
				t.Errorf("expected text %q, got %v", tt.text, textObj["text"])
			}
		})
	}
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

func TestRunHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.history", r.URL.Path)
		assert.Equal(t, "C123", r.URL.Query().Get("channel"))
		assert.Equal(t, "20", r.URL.Query().Get("limit"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"ts": "1234567890.123456", "user": "U001", "text": "Hello"},
				{"ts": "1234567890.123457", "user": "U002", "text": "World"},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	err := runHistory("C123", opts, c)
	require.NoError(t, err)
}

func TestRunHistory_WithTimeRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "1234567890.000000", r.URL.Query().Get("oldest"))
		assert.Equal(t, "1234567899.000000", r.URL.Query().Get("latest"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"messages": []map[string]interface{}{},
		})
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
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"messages": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &historyOptions{limit: 20}

	err := runHistory("C123", opts, c)
	require.NoError(t, err)
}

func TestRunThread_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.replies", r.URL.Path)
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
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &threadOptions{limit: 100}

	err := runThread("C123", "1234567890.123456", opts, c)
	require.NoError(t, err)
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

func TestRunSend_InvalidChannelID(t *testing.T) {
	opts := &sendOptions{simple: true}
	err := runSend("invalid", "Hello", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel ID")
}

func TestRunSend_InvalidThreadTimestamp(t *testing.T) {
	opts := &sendOptions{simple: true, threadTS: "not-a-timestamp"}
	err := runSend("C123456789", "Hello", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunDelete_InvalidChannelID(t *testing.T) {
	opts := &deleteOptions{force: true}
	err := runDelete("invalid", "1234567890.123456", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel ID")
}

func TestRunDelete_InvalidTimestamp(t *testing.T) {
	opts := &deleteOptions{force: true}
	err := runDelete("C123456789", "not-a-timestamp", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunReact_InvalidChannelID(t *testing.T) {
	opts := &reactOptions{}
	err := runReact("invalid", "1234567890.123456", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel ID")
}

func TestRunReact_InvalidTimestamp(t *testing.T) {
	opts := &reactOptions{}
	err := runReact("C123456789", "not-a-timestamp", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestRunUnreact_InvalidChannelID(t *testing.T) {
	opts := &unreactOptions{}
	err := runUnreact("invalid", "1234567890.123456", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid channel ID")
}

func TestRunUnreact_InvalidTimestamp(t *testing.T) {
	opts := &unreactOptions{}
	err := runUnreact("C123456789", "not-a-timestamp", "thumbsup", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
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
