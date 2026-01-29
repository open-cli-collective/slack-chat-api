package whoami

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

func TestRunWhoami_BotTokenOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth.test", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "test-bot",
			"bot_id":  "B123456",
			"team_id": "T123456",
			"user_id": "U123456",
		})
	}))
	defer server.Close()

	botClient := client.NewWithConfig(server.URL, "xoxb-test", nil)
	opts := &whoamiOptions{}

	// Pass bot client, nil for user client
	err := runWhoami(opts, botClient, nil)
	require.NoError(t, err)
}

func TestRunWhoami_UserTokenOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/auth.test", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "human-user",
			"team_id": "T123456",
			"user_id": "U789012",
		})
	}))
	defer server.Close()

	userClient := client.NewWithConfig(server.URL, "xoxp-test", nil)
	opts := &whoamiOptions{}

	// Pass nil for bot client, user client provided
	err := runWhoami(opts, nil, userClient)
	require.NoError(t, err)
}

func TestRunWhoami_BothTokens(t *testing.T) {
	botServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "test-bot",
			"bot_id":  "B123456",
			"team_id": "T123456",
			"user_id": "U123456",
		})
	}))
	defer botServer.Close()

	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "human-user",
			"team_id": "T123456",
			"user_id": "U789012",
		})
	}))
	defer userServer.Close()

	botClient := client.NewWithConfig(botServer.URL, "xoxb-test", nil)
	userClient := client.NewWithConfig(userServer.URL, "xoxp-test", nil)
	opts := &whoamiOptions{}

	err := runWhoami(opts, botClient, userClient)
	require.NoError(t, err)
}

func TestRunWhoami_NoTokens(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain may have stored token")
	}

	// Use temp dir with no token set
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("SLACK_API_TOKEN", "")
	t.Setenv("SLACK_USER_TOKEN", "")

	opts := &whoamiOptions{}

	// Pass nil clients to trigger token lookup
	err := runWhoami(opts, nil, nil)
	require.NoError(t, err)
}

func TestRunWhoami_AuthFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer server.Close()

	botClient := client.NewWithConfig(server.URL, "bad-token", nil)
	opts := &whoamiOptions{}

	// Auth fails, but function should handle gracefully
	err := runWhoami(opts, botClient, nil)
	require.NoError(t, err)
}

func TestRunWhoami_BotWithoutBotID(t *testing.T) {
	// Some tokens may not have a bot_id
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "legacy-bot",
			"team_id": "T123456",
			"user_id": "U123456",
		})
	}))
	defer server.Close()

	botClient := client.NewWithConfig(server.URL, "xoxb-test", nil)
	opts := &whoamiOptions{}

	err := runWhoami(opts, botClient, nil)
	require.NoError(t, err)
}
