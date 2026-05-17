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
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "team": "Test Workspace", "user": "testbot",
			"team_id": "T123", "user_id": "U123", "bot_id": "B123",
		})
	}))
}

func mockClient(s *httptest.Server) func(string, string) *client.Client {
	return func(_, token string) *client.Client { return client.NewWithConfig(s.URL, token, nil) }
}

func hasBot(t *testing.T) bool {
	t.Helper()
	st, err := keychain.Open()
	require.NoError(t, err)
	defer func() { _ = st.Close() }()
	return st.HasBotToken()
}

func TestRunInit_FromEnv_BotOnly(t *testing.T) {
	testutil.Setup(t)
	s := newMockServer(t)
	defer s.Close()
	t.Setenv("BOT_TOK", "xoxb-test-token-12345678")

	err := runInit(&initOptions{botEnv: "BOT_TOK", newClient: mockClient(s)})
	require.NoError(t, err)
	assert.True(t, hasBot(t))
}

func TestRunInit_FromEnv_BothTokens(t *testing.T) {
	testutil.Setup(t)
	s := newMockServer(t)
	defer s.Close()
	t.Setenv("BOT_TOK", "xoxb-test-token-12345678")
	t.Setenv("USR_TOK", "xoxp-test-token-12345678")

	err := runInit(&initOptions{botEnv: "BOT_TOK", userEnv: "USR_TOK", newClient: mockClient(s)})
	require.NoError(t, err)

	st, err := keychain.Open()
	require.NoError(t, err)
	defer func() { _ = st.Close() }()
	assert.True(t, st.HasBotToken())
	assert.True(t, st.HasUserToken())
}

func TestRunInit_Stdin_NoVerify(t *testing.T) {
	testutil.Setup(t)
	err := runInit(&initOptions{
		botStdin: true,
		stdin:    strings.NewReader("xoxb-test-token-12345678\n"),
		noVerify: true,
	})
	require.NoError(t, err)
	assert.True(t, hasBot(t))
}

// Fixes the B0-deferred non-hermetic flake: testutil.Setup forces the file
// backend in a temp HOME, so this no longer races the real macOS keychain.
func TestRunInit_WrongTokenType(t *testing.T) {
	testutil.Setup(t)
	err := runInit(&initOptions{
		botStdin: true,
		stdin:    strings.NewReader("xoxp-this-is-a-user-token\n"),
		noVerify: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected bot token")
}

func TestRunInit_WrongUserTokenType(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("BOT_TOK", "xoxb-test-token-12345678")
	t.Setenv("USR_TOK", "xoxb-this-is-a-bot-token")
	err := runInit(&initOptions{botEnv: "BOT_TOK", userEnv: "USR_TOK", noVerify: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected user token")
}

func TestRunInit_Interactive_NoTokensProvided(t *testing.T) {
	testutil.Setup(t)
	// empty bot prompt, then "n" to the add-user prompt.
	err := runInit(&initOptions{stdin: strings.NewReader("\nn\n"), noVerify: true})
	require.NoError(t, err)
	assert.False(t, hasBot(t))
}

func TestRunInit_Interactive_CancelOverwrite(t *testing.T) {
	testutil.Setup(t)
	st, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, st.SetBotToken("xoxb-existing-token"))
	_ = st.Close()

	// Existing creds + interactive + "n" to overwrite → cancelled.
	err = runInit(&initOptions{stdin: strings.NewReader("n\n"), noVerify: true})
	require.NoError(t, err)
}

func TestRunInit_VerificationFailed(t *testing.T) {
	testutil.Setup(t)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "invalid_auth"})
	}))
	defer s.Close()
	t.Setenv("BOT_TOK", "xoxb-bad-token-12345678")

	err := runInit(&initOptions{botEnv: "BOT_TOK", newClient: mockClient(s)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
}
