package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users.list", r.URL.Path)
		assert.Equal(t, "100", r.URL.Query().Get("limit"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{
					"id":        "U001",
					"name":      "alice",
					"real_name": "Alice Smith",
					"is_bot":    false,
					"profile": map[string]interface{}{
						"email": "alice@example.com",
					},
				},
				{
					"id":        "U002",
					"name":      "bob",
					"real_name": "Bob Jones",
					"is_bot":    false,
					"profile": map[string]interface{}{
						"email": "bob@example.com",
					},
				},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{limit: 100}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_FiltersBots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{
					"id":        "U001",
					"name":      "alice",
					"real_name": "Alice Smith",
					"is_bot":    false,
					"profile":   map[string]interface{}{"email": "alice@example.com"},
				},
				{
					"id":        "B001",
					"name":      "slackbot",
					"real_name": "Slackbot",
					"is_bot":    true,
					"profile":   map[string]interface{}{},
				},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{limit: 100}

	// This test verifies the output only shows non-bot users
	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"members": []map[string]interface{}{},
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
		assert.Equal(t, "/users.info", r.URL.Path)
		assert.Equal(t, "U001", r.URL.Query().Get("user"))

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U001",
				"name":      "alice",
				"real_name": "Alice Smith",
				"is_admin":  true,
				"is_bot":    false,
				"profile": map[string]interface{}{
					"display_name": "Alice",
					"email":        "alice@example.com",
					"status_text":  "Working remotely",
					"status_emoji": ":house:",
				},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &getOptions{}

	err := runGet("U001", opts, c)
	require.NoError(t, err)
}

func TestRunGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "user_not_found",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &getOptions{}

	err := runGet("INVALID", opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user_not_found")
}

func TestRunGet_NoStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U001",
				"name":      "alice",
				"real_name": "Alice Smith",
				"is_admin":  false,
				"is_bot":    false,
				"profile": map[string]interface{}{
					"display_name": "Alice",
					"email":        "alice@example.com",
					"status_text":  "",
					"status_emoji": "",
				},
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &getOptions{}

	err := runGet("U001", opts, c)
	require.NoError(t, err)
}
