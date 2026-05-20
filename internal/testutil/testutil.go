// Package testutil provides a hermetic credential environment for tests
// (§1.12 test obligation). It delegates state-dir isolation to the shared
// cli-common/statedirtest helper (the full 7-var env set per §3.1 — closes
// the Windows real-dir leak the old HOME/XDG-only setup had), then layers
// the slck-specific keyring backend selection on top: credstore's
// encrypted-file backend with a known passphrase, plus the legacy
// `security` scan disabler so no test ever shells out to the real OS
// keychain. This is the test pattern the B2/B3 migration tests reuse.
package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// Setup isolates the full §3.1 7-var env set under t.TempDir() (via
// statedirtest.Hermetic) and forces credstore's file backend with a known
// passphrase via the §1.4 named env vars. The darwin `security` legacy
// probe is neutralized so the suite is hermetic regardless of the
// destination backend (§2.4). Returns the temp root so a test can plant
// legacy artifacts — but tests should resolve their paths through
// ConfigDir(t) / LegacyCredentialsPath(t) below, not by hand-building
// subdirs, because os.UserConfigDir is platform-native (macOS ~/Library/
// Application Support, Windows %APPDATA%) and not derived from any single
// env var.
func Setup(t *testing.T) string {
	t.Helper()
	tmp := statedirtest.Hermetic(t)
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

// ConfigDir resolves the post-statedirtest hermetic CANONICAL config dir
// (statedir-resolved per OS) and creates it. Tests that plant or inspect
// `config.yml` should use this rather than hand-building subdirs of Setup's
// tmp root.
func ConfigDir(t *testing.T) string {
	t.Helper()
	dir, err := appconfig.Dir()
	if err != nil {
		t.Fatalf("testutil.ConfigDir: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("testutil.ConfigDir mkdir: %v", err)
	}
	return dir
}

// LegacyCredentialsPath returns the pre-MON-5372 hand-rolled `credentials`
// file path the keychain migrator's `legacyCredentialsPath()` scans:
// $XDG_CONFIG_HOME/slack-chat-api/credentials else $HOME/.config/...
// Distinct from ConfigDir on macOS/Windows where the canonical resolver
// returns a different OS-native path. Tests that seed/inspect the legacy
// credentials file (the secret-bearing k=v fixture, NOT config.yml) must
// use this helper. Returns the absolute path; the parent dir is created so
// tests can `os.WriteFile` immediately.
func LegacyCredentialsPath(t *testing.T) string {
	t.Helper()
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configHome, "slack-chat-api")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("testutil.LegacyCredentialsPath mkdir: %v", err)
	}
	return filepath.Join(dir, "credentials")
}
