package me

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

func TestRunMe_BotTokenOnly(t *testing.T) {
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
	opts := &meOptions{}

	// Pass bot client, nil for user client
	err := runMe(opts, botClient, nil)
	require.NoError(t, err)
}

func TestRunMe_UserTokenOnly(t *testing.T) {
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
	opts := &meOptions{}

	// Pass nil for bot client, user client provided
	err := runMe(opts, nil, userClient)
	require.NoError(t, err)
}

func TestRunMe_BothTokens(t *testing.T) {
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
	opts := &meOptions{}

	err := runMe(opts, botClient, userClient)
	require.NoError(t, err)
}

func TestRunMe_NoTokens(t *testing.T) {
	// Hermetic empty keyring (file backend, temp HOME) — no real keychain.
	testutil.Setup(t)

	opts := &meOptions{}

	// Pass nil clients to trigger token lookup
	err := runMe(opts, nil, nil)
	require.NoError(t, err)
}

func TestRunMe_AuthFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer server.Close()

	botClient := client.NewWithConfig(server.URL, "bad-token", nil)
	opts := &meOptions{}

	// Auth fails, but function should handle gracefully
	err := runMe(opts, botClient, nil)
	require.NoError(t, err)
}

func TestRunMe_BotWithoutBotID(t *testing.T) {
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
	opts := &meOptions{}

	err := runMe(opts, botClient, nil)
	require.NoError(t, err)
}
