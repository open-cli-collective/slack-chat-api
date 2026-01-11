package messages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piekstra/slack-cli/internal/client"
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
