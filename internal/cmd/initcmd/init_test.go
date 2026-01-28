package initcmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok":      true,
			"team":    "Test Workspace",
			"user":    "testbot",
			"team_id": "T123",
			"user_id": "U123",
			"bot_id":  "B123",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestRunInit_NonInteractive_BotOnly(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain can't be easily mocked")
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	server := newMockServer(t)
	defer server.Close()

	opts := &initOptions{
		botToken: "xoxb-test-token-12345678",
		newClient: func(_, token string) *client.Client {
			return client.NewWithConfig(server.URL, token, nil)
		},
	}

	err := runInit(opts)
	require.NoError(t, err)

	// Verify token was stored
	assert.True(t, keychain.HasStoredToken())
}

func TestRunInit_NonInteractive_BothTokens(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain can't be easily mocked")
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	server := newMockServer(t)
	defer server.Close()

	opts := &initOptions{
		botToken:  "xoxb-test-token-12345678",
		userToken: "xoxp-test-token-12345678",
		newClient: func(_, token string) *client.Client {
			return client.NewWithConfig(server.URL, token, nil)
		},
	}

	err := runInit(opts)
	require.NoError(t, err)

	assert.True(t, keychain.HasStoredToken())
	assert.True(t, keychain.HasStoredUserToken())
}

func TestRunInit_NonInteractive_NoVerify(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain can't be easily mocked")
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	opts := &initOptions{
		botToken: "xoxb-test-token-12345678",
		noVerify: true,
	}

	err := runInit(opts)
	require.NoError(t, err)

	assert.True(t, keychain.HasStoredToken())
}

func TestRunInit_WrongTokenType(t *testing.T) {
	opts := &initOptions{
		botToken: "xoxp-this-is-a-user-token",
		noVerify: true,
	}

	err := runInit(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected bot token")
}

func TestRunInit_WrongUserTokenType(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain can't be easily mocked")
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	opts := &initOptions{
		botToken:  "xoxb-test-token-12345678",
		userToken: "xoxb-this-is-a-bot-token",
		noVerify:  true,
	}

	err := runInit(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected user token")
}

func TestRunInit_Interactive_NoTokensProvided(t *testing.T) {
	// Simulate pressing enter (empty) for bot token, then "n" for user token prompt
	opts := &initOptions{
		stdin:    strings.NewReader("\nn\n"),
		noVerify: true,
	}

	err := runInit(opts)
	require.NoError(t, err)
}

func TestRunInit_Interactive_CancelOverwrite(t *testing.T) {
	if keychain.IsSecureStorage() {
		t.Skip("Skipping on macOS - keychain can't be easily mocked")
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Set an existing token first
	require.NoError(t, keychain.SetAPIToken("xoxb-existing-token"))

	// Simulate "n" for overwrite prompt
	opts := &initOptions{
		stdin:    strings.NewReader("n\n"),
		noVerify: true,
	}

	err := runInit(opts)
	require.NoError(t, err)
}

func TestRunInit_VerificationFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok":    false,
			"error": "invalid_auth",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	opts := &initOptions{
		botToken: "xoxb-bad-token-12345678",
		newClient: func(_, token string) *client.Client {
			return client.NewWithConfig(server.URL, token, nil)
		},
	}

	err := runInit(opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
}
