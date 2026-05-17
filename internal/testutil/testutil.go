// Package testutil provides a hermetic credential environment for tests
// (§1.12 test obligation). It forces credstore's encrypted-file backend
// inside a per-test temp HOME with a fixed passphrase, so no test ever
// touches the real OS keyring, shells out to `security`, or depends on
// machine state. This is the test pattern B2/B3 reuse.
package testutil

import (
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// Setup isolates HOME/XDG to a temp dir and forces the file backend with a
// known passphrase via the §1.4 named env vars. Returns the temp dir so a
// test can plant legacy artifacts (e.g. a legacy credentials file) under it.
func Setup(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdgconfig"))
	// Force credstore's encrypted-file backend (never the real Keychain /
	// Secret Service), with the passphrase supplied non-interactively.
	t.Setenv("SLACK_CHAT_API_KEYRING_BACKEND", "file")
	t.Setenv("SLACK_CHAT_API_KEYRING_PASSPHRASE", "test-passphrase")
	// Neutralize the darwin legacy-Keychain `security` probe so the suite
	// is hermetic. This is independent of the destination backend: the
	// migration's discovery matrix (§2.4) otherwise always runs on macOS.
	t.Setenv("SLCK_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN", "1")
	// Belt-and-suspenders: a prior test's recorded §1.8 block must never
	// bleed into this one's JSON output.
	output.ResetMigration()
	t.Cleanup(output.ResetMigration)
	return tmp
}
