package keychain

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

const (
	botTok  = "xoxb-1111-distinctive-bot-secret"
	userTok = "xoxp-2222-distinctive-user-secret"
)

// TestPlanMigration exercises the pure §1.8 resolver directly — including
// the legacy-vs-legacy disagreement branch, which the file/keychain
// plumbing can't reach (a key=value file yields one value per key and the
// darwin Keychain scan is disabled in tests).
func TestPlanMigration(t *testing.T) {
	cand := func(val, loc string) candidate {
		return candidate{newKey: KeyBotToken, legacyField: "api_token", value: val,
			location: loc, deleter: func() error { return nil }}
	}
	none := func(string) (string, bool) { return "", false }
	has := func(v string) func(string) (string, bool) {
		return func(string) (string, bool) { return v, true }
	}

	t.Run("legacy-vs-legacy disagreement is always a conflict", func(t *testing.T) {
		for _, ow := range []bool{false, true} { // overwrite cannot resolve it
			_, err := planMigration("slack-chat-api", "default", "slack-chat-api/default",
				[]candidate{cand("xoxb-AAA", "keychain:slck/api_token"),
					cand("xoxb-BBB", "file:/p/credentials#api_token")}, none, ow)
			if !errors.Is(err, credstore.ErrMigrationConflict) {
				t.Fatalf("overwrite=%v: want ErrMigrationConflict, got %v", ow, err)
			}
			if leak := credstore.NoLeakAssertion([]byte(err.Error()),
				"xoxb-AAA", "xoxb-BBB"); leak != nil {
				t.Fatalf("conflict leaked a value: %v", leak)
			}
			if !strings.Contains(err.Error(), "keychain:slck/api_token") ||
				!strings.Contains(err.Error(), "file:/p/credentials#api_token") {
				t.Fatalf("conflict must name all sources: %q", err.Error())
			}
		}
	})

	t.Run("legacy-vs-target differs, no overwrite -> conflict", func(t *testing.T) {
		_, err := planMigration("svc", "default", "svc/default",
			[]candidate{cand("xoxb-LEGACY", "file:x#api_token")}, has("xoxb-TARGET"), false)
		if !errors.Is(err, credstore.ErrMigrationConflict) {
			t.Fatalf("want conflict, got %v", err)
		}
	})

	t.Run("legacy-vs-target differs, overwrite -> write", func(t *testing.T) {
		p, err := planMigration("svc", "default", "svc/default",
			[]candidate{cand("xoxb-LEGACY", "file:x#api_token")}, has("xoxb-TARGET"), true)
		if err != nil || p.writes[KeyBotToken] != "xoxb-LEGACY" || len(p.changes) != 1 {
			t.Fatalf("overwrite should force legacy: plan=%+v err=%v", p, err)
		}
	})

	t.Run("equal target -> cleanup only, no write/signal", func(t *testing.T) {
		p, err := planMigration("svc", "default", "svc/default",
			[]candidate{cand("xoxb-SAME", "file:x#api_token")}, has("xoxb-SAME"), false)
		if err != nil || len(p.writes) != 0 || len(p.changes) != 0 || len(p.cleanups) != 1 {
			t.Fatalf("equal value should clean up silently: plan=%+v err=%v", p, err)
		}
	})

	t.Run("identical duplicate legacy values -> single write", func(t *testing.T) {
		p, err := planMigration("svc", "default", "svc/default",
			[]candidate{cand("xoxb-DUP", "keychain:slck/api_token"),
				cand("xoxb-DUP", "file:x#api_token")}, none, false)
		if err != nil || p.writes[KeyBotToken] != "xoxb-DUP" || len(p.cleanups) != 2 {
			t.Fatalf("byte-identical dup should migrate once: plan=%+v err=%v", p, err)
		}
	})
}

func TestDetectTokenType(t *testing.T) {
	cases := map[string]string{
		"xoxb-abc": "bot", "xoxp-abc": "user", "xoxc-abc": "unknown", "": "unknown",
	}
	for in, want := range cases {
		if got := DetectTokenType(in); got != want {
			t.Fatalf("DetectTokenType(%q)=%q want %q", in, got, want)
		}
	}
}

// TestRuntimeEnvNotReadAsCredential locks the headline §1.11 invariant:
// SLACK_API_TOKEN / SLACK_USER_TOKEN must NOT be consulted at runtime, even
// when set. With an empty keyring, reads must still fail-missing.
func TestRuntimeEnvNotReadAsCredential(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("SLACK_API_TOKEN", "xoxb-env-must-be-ignored")
	t.Setenv("SLACK_USER_TOKEN", "xoxp-env-must-be-ignored")

	st, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	if _, err := st.BotToken(); !errors.Is(err, ErrMissingBotToken) {
		t.Fatalf("SLACK_API_TOKEN was consulted at runtime (err=%v)", err)
	}
	if _, err := st.UserToken(); !errors.Is(err, ErrMissingUserToken) {
		t.Fatalf("SLACK_USER_TOKEN was consulted at runtime (err=%v)", err)
	}
	if st.HasBotToken() || st.HasUserToken() {
		t.Fatal("Has*Token true purely from env — §1.11 violation")
	}
}

func TestStoreRoundTripAndClear(t *testing.T) {
	testutil.Setup(t)
	st, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()

	if _, err := st.BotToken(); !errors.Is(err, ErrMissingBotToken) {
		t.Fatalf("missing bot token err=%v want ErrMissingBotToken", err)
	}
	if err := st.SetBotToken(botTok); err != nil {
		t.Fatalf("SetBotToken: %v", err)
	}
	if err := st.SetUserToken(userTok); err != nil {
		t.Fatalf("SetUserToken: %v", err)
	}
	if v, err := st.BotToken(); err != nil || v != botTok {
		t.Fatalf("BotToken=%q,%v", v, err)
	}
	if !st.HasBotToken() || !st.HasUserToken() {
		t.Fatalf("Has*Token false after set")
	}
	if err := st.DeleteUserToken(); err != nil {
		t.Fatalf("DeleteUserToken: %v", err)
	}
	if st.HasUserToken() {
		t.Fatalf("user token still present after delete")
	}
	if err := st.DeleteUserToken(); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
	removed, err := st.Clear()
	if err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if len(removed) == 0 || st.HasBotToken() {
		t.Fatalf("Clear left state: removed=%v", removed)
	}
}

// TestRefAuthoritative proves nothing is hard-coded: a non-default ref in
// config.yml drives the service/profile, and a value written under it is not
// visible under the default ref (§1.3 — Codex Blocker).
func TestRefAuthoritative(t *testing.T) {
	testutil.Setup(t)
	st, err := openWith(&appconfig.Config{CredentialRef: "slack-chat-api/work"}, false, true)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	if st.Ref() != "slack-chat-api/work" {
		t.Fatalf("Ref=%q", st.Ref())
	}
	if err := st.SetBotToken(botTok); err != nil {
		t.Fatalf("set: %v", err)
	}
	_ = st.Close()

	def, err := openWith(&appconfig.Config{CredentialRef: appconfig.DefaultCredentialRef}, false, true)
	if err != nil {
		t.Fatalf("open default: %v", err)
	}
	defer func() { _ = def.Close() }()
	if def.HasBotToken() {
		t.Fatalf("value leaked across profiles")
	}
}

func writeLegacyFile(t *testing.T, kv map[string]string) string {
	t.Helper()
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "slack-chat-api")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "credentials")
	var b strings.Builder
	for k, v := range kv {
		b.WriteString(k + "=" + v + "\n")
	}
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMigratePlaintextFileRenamesAndCleansUp(t *testing.T) {
	testutil.Setup(t)
	legacy := writeLegacyFile(t, map[string]string{"api_token": botTok, "user_token": userTok})

	st, err := Open()
	if err != nil {
		t.Fatalf("Open(migrate): %v", err)
	}
	defer func() { _ = st.Close() }()

	if v, _ := st.BotToken(); v != botTok {
		t.Fatalf("api_token did not migrate to bot_token: %q", v)
	}
	if v, _ := st.UserToken(); v != userTok {
		t.Fatalf("user_token did not migrate: %q", v)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy plaintext file not removed: %v", err)
	}
	if _, err := os.Stat(appconfig.Path()); err != nil {
		t.Fatalf("config.yml not written: %v", err)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	testutil.Setup(t)
	writeLegacyFile(t, map[string]string{"api_token": botTok})
	s1, err := Open()
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	_ = s1.Close()
	output.ResetMigration()

	s2, err := Open()
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer func() { _ = s2.Close() }()
	if v, _ := s2.BotToken(); v != botTok {
		t.Fatalf("value lost on idempotent re-open: %q", v)
	}
}

func TestMigrateConflictFailsLoudWithoutLeaking(t *testing.T) {
	testutil.Setup(t)
	pre, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := pre.SetBotToken("xoxb-KEYRING-existing-value"); err != nil {
		t.Fatal(err)
	}
	_ = pre.Close()

	legacy := writeLegacyFile(t, map[string]string{"api_token": "xoxb-LEGACY-different-value"})

	_, err = Open()
	if !errors.Is(err, credstore.ErrMigrationConflict) {
		t.Fatalf("want ErrMigrationConflict, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, legacy) || !strings.Contains(msg, "slack-chat-api/default") {
		t.Fatalf("conflict msg missing locations: %q", msg)
	}
	if leak := credstore.NoLeakAssertion([]byte(msg),
		"xoxb-KEYRING-existing-value", "xoxb-LEGACY-different-value"); leak != nil {
		t.Fatalf("conflict message leaked a secret value: %v", leak)
	}
	if _, statErr := os.Stat(legacy); statErr != nil {
		t.Fatalf("legacy file deleted despite conflict: %v", statErr)
	}
}

func TestMigrateConflictResolvedByOverwrite(t *testing.T) {
	testutil.Setup(t)
	pre, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := pre.SetBotToken("xoxb-OLD-keyring"); err != nil {
		t.Fatal(err)
	}
	_ = pre.Close()
	legacy := writeLegacyFile(t, map[string]string{"api_token": "xoxb-NEW-legacy-forced"})

	st, err := OpenForMigrationOverwrite()
	if err != nil {
		t.Fatalf("overwrite migrate: %v", err)
	}
	defer func() { _ = st.Close() }()
	if v, _ := st.BotToken(); v != "xoxb-NEW-legacy-forced" {
		t.Fatalf("overwrite did not force legacy: %q", v)
	}
	if _, e := os.Stat(legacy); !os.IsNotExist(e) {
		t.Fatalf("legacy not removed after forced migrate")
	}
}

func TestMigrateEqualValueCleansUpSilently(t *testing.T) {
	testutil.Setup(t)
	pre, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := pre.SetBotToken(botTok); err != nil {
		t.Fatal(err)
	}
	_ = pre.Close()
	legacy := writeLegacyFile(t, map[string]string{"api_token": botTok})
	output.ResetMigration()

	st, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = st.Close() }()
	if _, e := os.Stat(legacy); !os.IsNotExist(e) {
		t.Fatalf("equal-value legacy not cleaned up")
	}
	if v, _ := st.BotToken(); v != botTok {
		t.Fatalf("value changed: %q", v)
	}
}

func TestDiscoverFileBranch(t *testing.T) {
	testutil.Setup(t)
	writeLegacyFile(t, map[string]string{"api_token": botTok, "ignored_key": "x"})
	got := discover("slack-chat-api")
	var sawBot bool
	for _, c := range got {
		if c.newKey == KeyBotToken {
			sawBot = true
			if c.legacyField != "api_token" {
				t.Fatalf("legacyField=%q want api_token", c.legacyField)
			}
		}
		if c.legacyField == "ignored_key" {
			t.Fatalf("non-credential key discovered")
		}
	}
	if !sawBot {
		t.Fatalf("api_token not discovered from file: %+v", got)
	}
}
