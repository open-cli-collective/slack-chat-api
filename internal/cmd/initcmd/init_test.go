package initcmd

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

// writeLegacyCreds writes the legacy plaintext credentials file at the path
// the keychain migrator's legacyCredentialsPath() scans (the hand-rolled
// XDG/$HOME/.config path — distinct from the new canonical config dir on
// macOS/Windows post-MON-5372).
func writeLegacyCreds(t *testing.T, kv map[string]string) string {
	t.Helper()
	path := testutil.LegacyCredentialsPath(t)
	var b strings.Builder
	for k, v := range kv {
		b.WriteString(k + "=" + v + "\n")
	}
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o600))
	return path
}

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

// A legacy value that disagrees with an existing keyring value must, WITHOUT
// --overwrite, fail loud (§1.8) and never leak either secret — exercised
// through the init command's default Open path, not just the resolver.
func TestRunInit_MigrationConflict_FailsLoudWithoutLeaking(t *testing.T) {
	testutil.Setup(t)
	pre, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, pre.SetBotToken("xoxb-OLD-keyring-value"))
	_ = pre.Close()
	writeLegacyCreds(t, map[string]string{"api_token": "xoxb-NEW-legacy-value"})

	err = runInit(&initOptions{stdin: strings.NewReader("\nn\n"), noVerify: true})
	require.Error(t, err)
	assert.True(t, errors.Is(err, credstore.ErrMigrationConflict),
		"want ErrMigrationConflict, got %v", err)
	assert.NotContains(t, err.Error(), "xoxb-OLD-keyring-value")
	assert.NotContains(t, err.Error(), "xoxb-NEW-legacy-value")
}

// `slck init --overwrite` is the only path that forces a legacy value over a
// conflicting keyring entry. Exercise the CLI flag → OpenForMigrationOverwrite
// wiring end to end (the resolver itself is unit-tested separately).
func TestRunInit_OverwriteResolvesMigrationConflict(t *testing.T) {
	testutil.Setup(t)
	pre, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, pre.SetBotToken("xoxb-OLD-keyring-value"))
	_ = pre.Close()
	legacy := writeLegacyCreds(t, map[string]string{"api_token": "xoxb-NEW-legacy-value"})

	// No tokens supplied: init resolves the conflict during Open, then exits
	// on "no tokens provided" — the migration side effect is what we assert.
	err = runInit(&initOptions{overwrite: true, stdin: strings.NewReader("\nn\n"), noVerify: true})
	require.NoError(t, err)

	st, err := keychain.OpenNoMigrate()
	require.NoError(t, err)
	defer func() { _ = st.Close() }()
	v, err := st.BotToken()
	require.NoError(t, err)
	assert.Equal(t, "xoxb-NEW-legacy-value", v, "--overwrite must force the legacy value")
	_, statErr := os.Stat(legacy)
	assert.True(t, os.IsNotExist(statErr), "legacy file must be removed after forced migrate")
}

// TestRunInit_RelocationGate_OldOnlyCopied pins the MON-5372 init-level
// contract: when only the old hand-rolled config.yml exists, runInit must
// copy it to the new canonical dir BEFORE any token migration / save runs.
// Tests the actual gate ordering through runInit (not just the unit-level
// Detect+Apply pair), so a regression that removed or reordered the gate
// would surface here.
func TestRunInit_RelocationGate_OldOnlyCopied(t *testing.T) {
	testutil.Setup(t)
	// On Linux old == new path; the gate is a no-op and the test simply
	// confirms init still works. On macOS/Windows these diverge and the
	// gate actually performs the copy.
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		// statedirtest sets it; defensive.
		t.Skip("hermetic env did not set XDG_CONFIG_HOME")
	}
	oldDir := filepath.Join(configHome, "slack-chat-api")
	require.NoError(t, os.MkdirAll(oldDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "config.yml"),
		[]byte("credential_ref: slack-chat-api/from-old\nworkspace: T_OLD_PRE_INIT\n"), 0o600))

	// Run init with no tokens — wizard short-circuits at "no tokens provided"
	// AFTER the relocation gate has run (and AFTER EnsureMigrated). The gate
	// must have copied old→new by then if they differ.
	err := runInit(&initOptions{stdin: strings.NewReader("\nn\n"), noVerify: true})
	require.NoError(t, err)

	// New canonical path must exist with the old content (or, on Linux,
	// remain at the same path — same assertion either way).
	newDir, err := appconfig.Dir()
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(newDir, "config.yml"))
	require.NoError(t, err, "new canonical config.yml must exist after init")
	assert.Contains(t, string(data), "slack-chat-api/from-old",
		"new canonical must carry the old config content")
}

// TestRunInit_RelocationGate_DivergentAbortsBeforeMutation pins the MON-5372
// fail-loud contract through runInit: divergent old/new config aborts init
// BEFORE any token migration or config write papers over the conflict. Skips
// on Linux where old == new path (the gate short-circuits to relocNone).
func TestRunInit_RelocationGate_DivergentAbortsBeforeMutation(t *testing.T) {
	testutil.Setup(t)
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		t.Skip("hermetic env did not set XDG_CONFIG_HOME")
	}
	// On Linux the resolved new dir IS the XDG-rooted dir — old == new, no
	// way to construct divergence at the public seam. The unit-level test
	// already covers this branch on every OS via the injectable newDir.
	newDir, err := appconfig.Dir()
	require.NoError(t, err)
	oldDir := filepath.Join(configHome, "slack-chat-api")
	if newDir == oldDir {
		t.Skip("Linux: old path equals new path; divergence covered at unit level")
	}

	require.NoError(t, os.MkdirAll(oldDir, 0o700))
	require.NoError(t, os.MkdirAll(newDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "config.yml"),
		[]byte("credential_ref: slack-chat-api/old\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "config.yml"),
		[]byte("credential_ref: slack-chat-api/new\n"), 0o600))

	// Seed a legacy credentials file too. If the relocation gate were
	// accidentally moved AFTER keychain.Open, the migrator would consume
	// this file before the abort and the post-abort stat would fail. With
	// the gate ordered correctly, the legacy file stays untouched.
	legacy := writeLegacyCreds(t, map[string]string{"api_token": "xoxb-MUST-NOT-BE-MIGRATED"})

	// Snapshot pre-init bytes to assert mutate-nothing.
	oldBefore, _ := os.ReadFile(filepath.Join(oldDir, "config.yml"))
	newBefore, _ := os.ReadFile(filepath.Join(newDir, "config.yml"))

	err = runInit(&initOptions{botEnv: "BOT_TOK", noVerify: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "detecting config relocation",
		"init must abort at the relocation gate")

	// Nothing mutated by the abort.
	oldAfter, _ := os.ReadFile(filepath.Join(oldDir, "config.yml"))
	newAfter, _ := os.ReadFile(filepath.Join(newDir, "config.yml"))
	assert.Equal(t, string(oldBefore), string(oldAfter), "old config must not be mutated by aborted init")
	assert.Equal(t, string(newBefore), string(newAfter), "new config must not be mutated by aborted init")

	// Critical proof the gate ran BEFORE keychain.Open / migration: the
	// legacy credentials file must still exist. If migration had run, it
	// would have consumed and removed this file.
	if _, statErr := os.Stat(legacy); statErr != nil {
		t.Errorf("legacy credentials file must still exist (gate must abort before migration); stat err=%v", statErr)
	}
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
