package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

func TestRunTest_Success(t *testing.T) {
	tests := []struct {
		name     string
		response map[string]interface{}
	}{
		{
			name: "bot token with bot_id",
			response: map[string]interface{}{
				"ok":      true,
				"team":    "Test Workspace",
				"user":    "test-bot",
				"bot_id":  "B123456",
				"team_id": "T123456",
				"user_id": "U123456",
			},
		},
		{
			name: "user token without bot_id",
			response: map[string]interface{}{
				"ok":      true,
				"team":    "Another Workspace",
				"user":    "human-user",
				"team_id": "T789",
				"user_id": "U789",
			},
		},
		{
			name: "minimal response",
			response: map[string]interface{}{
				"ok":   true,
				"team": "Minimal",
				"user": "bot",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/auth.test", r.URL.Path)
				assert.Equal(t, "POST", r.Method)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			c := client.NewWithConfig(server.URL, "test-token", nil)
			opts := &testOptions{}

			// Pass same client for both bot and user for simplicity in test
			err := runTest(opts, c, c)
			require.NoError(t, err)
		})
	}
}

func TestRunTest_AuthErrors(t *testing.T) {
	// Note: The runTest function handles auth errors gracefully by printing
	// failure messages but not returning errors. This allows it to test
	// multiple tokens and report all results.
	tests := []struct {
		name     string
		response map[string]interface{}
	}{
		{
			name: "invalid_auth",
			response: map[string]interface{}{
				"ok":    false,
				"error": "invalid_auth",
			},
		},
		{
			name: "token_revoked",
			response: map[string]interface{}{
				"ok":    false,
				"error": "token_revoked",
			},
		},
		{
			name: "account_inactive",
			response: map[string]interface{}{
				"ok":    false,
				"error": "account_inactive",
			},
		},
		{
			name: "missing_scope",
			response: map[string]interface{}{
				"ok":    false,
				"error": "missing_scope",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			c := client.NewWithConfig(server.URL, "bad-token", nil)
			opts := &testOptions{}

			// Pass same client for both bot and user for simplicity in test
			// Function should not return error - it prints failures and continues
			err := runTest(opts, c, c)
			require.NoError(t, err)
		})
	}
}

func TestRunTest_NetworkErrors(t *testing.T) {
	// Note: The runTest function handles errors gracefully by printing
	// failure messages but not returning errors. This allows it to test
	// multiple tokens and report all results.

	t.Run("server unavailable", func(t *testing.T) {
		// Use a port that's not listening
		c := client.NewWithConfig("http://localhost:59999", "test-token", nil)
		opts := &testOptions{}

		// Pass same client for both bot and user for simplicity in test
		// Function should not return error - it prints failures and continues
		err := runTest(opts, c, c)
		require.NoError(t, err)
	})

	t.Run("server returns 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		c := client.NewWithConfig(server.URL, "test-token", nil)
		opts := &testOptions{}

		// Pass same client for both bot and user for simplicity in test
		// Function should not return error - it prints failures and continues
		err := runTest(opts, c, c)
		require.NoError(t, err)
	})

	t.Run("server returns invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		c := client.NewWithConfig(server.URL, "test-token", nil)
		opts := &testOptions{}

		// Pass same client for both bot and user for simplicity in test
		// Function should not return error - it prints failures and continues
		err := runTest(opts, c, c)
		require.NoError(t, err)
	})
}

func TestRunTest_NoTokenConfigured(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain may have stored token")
	}

	// Use temp dir with no token set
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("SLACK_API_TOKEN", "")  // Ensure bot env var is empty
	t.Setenv("SLACK_USER_TOKEN", "") // Ensure user env var is empty

	opts := &testOptions{}

	// Pass nil clients to trigger token lookup - should report "not configured" for both
	err := runTest(opts, nil, nil)
	// The function should return nil since it handles missing tokens gracefully
	require.NoError(t, err)
}
