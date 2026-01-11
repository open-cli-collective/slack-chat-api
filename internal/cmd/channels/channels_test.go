package channels

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piekstra/slack-cli/internal/client"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.list", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("exclude_archived"))
		assert.Equal(t, "100", r.URL.Query().Get("limit"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C123", "name": "general", "num_members": 10, "is_private": false},
				{"id": "C456", "name": "random", "num_members": 5, "is_private": false},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{limit: 100, excludeArchived: true}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_WithTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "public_channel,private_channel", r.URL.Query().Get("types"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{types: "public_channel,private_channel", limit: 100}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{limit: 100}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{limit: 100}

	err := runList(opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_auth")
}

func TestRunGet_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.info", r.URL.Path)
		assert.Equal(t, "C123", r.URL.Query().Get("channel"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":          "C123",
				"name":        "general",
				"is_private":  false,
				"is_archived": false,
				"num_members": 42,
				"topic":       map[string]interface{}{"value": "General discussion"},
				"purpose":     map[string]interface{}{"value": "Company-wide announcements"},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &getOptions{}

	err := runGet("C123", opts, c)
	require.NoError(t, err)
}

func TestRunGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "channel_not_found",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &getOptions{}

	err := runGet("INVALID", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "channel_not_found")
}

func TestRunCreate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.create", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "new-channel", body["name"])
		assert.Equal(t, false, body["is_private"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C789",
				"name": "new-channel",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &createOptions{private: false}

	err := runCreate("new-channel", opts, c)
	require.NoError(t, err)
}

func TestRunCreate_Private(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, true, body["is_private"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "G789",
				"name": "private-channel",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &createOptions{private: true}

	err := runCreate("private-channel", opts, c)
	require.NoError(t, err)
}

func TestRunArchive_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.archive", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &archiveOptions{}

	err := runArchive("C123", opts, c)
	require.NoError(t, err)
}

func TestRunUnarchive_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.unarchive", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &unarchiveOptions{}

	err := runUnarchive("C123", opts, c)
	require.NoError(t, err)
}

func TestRunSetTopic_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.setTopic", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "New topic", body["topic"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &setTopicOptions{}

	err := runSetTopic("C123", "New topic", opts, c)
	require.NoError(t, err)
}

func TestRunSetPurpose_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.setPurpose", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "New purpose", body["purpose"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &setPurposeOptions{}

	err := runSetPurpose("C123", "New purpose", opts, c)
	require.NoError(t, err)
}

func TestRunInvite_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.invite", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "C123", body["channel"])
		assert.Equal(t, "U001,U002", body["users"])

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":   "C123",
				"name": "general",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &inviteOptions{}

	err := runInvite("C123", []string{"U001", "U002"}, opts, c)
	require.NoError(t, err)
}

// Confirmation prompt tests for archive command

func TestRunArchive_Confirmation(t *testing.T) {
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
			opts := &archiveOptions{
				force: tt.force,
				stdin: strings.NewReader(tt.input),
			}

			err := runArchive("C123456789", opts, c)
			require.NoError(t, err)
			assert.Equal(t, tt.expectAPICall, apiCalled, "API call expectation mismatch")
		})
	}
}

// Validation tests for archive command

func TestRunArchive_InvalidChannelID(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		wantErr   string
	}{
		{"invalid prefix", "D123456789", "invalid channel ID"},
		{"lowercase prefix", "c123456789", "invalid channel ID"},
		{"empty string", "", "invalid channel ID"},
		{"just prefix", "C", "invalid channel ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &archiveOptions{force: true}
			// Pass nil client - validation should fail before client is needed
			err := runArchive(tt.channelID, opts, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
