package keychain

import (
	"errors"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/credstore"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

// TestOpenWith_ConfigOnlyMemoryBackend proves that with no --backend
// override, openWith routes config.Keyring.Backend through credstore's
// in-process memory backend and reports SourceConfig.
func TestOpenWith_ConfigOnlyMemoryBackend(t *testing.T) {
	testutil.Setup(t)
	// testutil pins SLACK_CHAT_API_KEYRING_BACKEND=file; clear it so the
	// config-side value wins.
	t.Setenv("SLACK_CHAT_API_KEYRING_BACKEND", "")
	SetBackendFlagOverride("", false)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	cfg := &appconfig.Config{
		CredentialRef: appconfig.DefaultCredentialRef,
		Keyring:       appconfig.KeyringConfig{Backend: string(credstore.BackendMemory)},
	}
	s, err := openWith(cfg, false, false)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	defer func() { _ = s.Close() }()

	backend, src := s.Backend()
	if backend != credstore.BackendMemory {
		t.Errorf("backend = %q, want %q", backend, credstore.BackendMemory)
	}
	if src != credstore.SourceConfig {
		t.Errorf("source = %q, want %q", src, credstore.SourceConfig)
	}
}

// TestOpenWith_FlagOverridesConfig proves the precedence: flag wins over
// config. Config asks for file, flag asks for memory → memory + Explicit.
func TestOpenWith_FlagOverridesConfig(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("SLACK_CHAT_API_KEYRING_BACKEND", "")
	SetBackendFlagOverride(string(credstore.BackendMemory), true)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	cfg := &appconfig.Config{
		CredentialRef: appconfig.DefaultCredentialRef,
		Keyring:       appconfig.KeyringConfig{Backend: string(credstore.BackendFile)},
	}
	s, err := openWith(cfg, false, false)
	if err != nil {
		t.Fatalf("openWith: %v", err)
	}
	defer func() { _ = s.Close() }()

	backend, src := s.Backend()
	if backend != credstore.BackendMemory {
		t.Errorf("backend = %q, want %q", backend, credstore.BackendMemory)
	}
	if src != credstore.SourceExplicit {
		t.Errorf("source = %q, want %q", src, credstore.SourceExplicit)
	}
}

// TestOpenWith_InvalidConfigBackend_FailsClosed drives an invalid
// keyring.backend value through openWith end-to-end. Because
// credstore.BindBackendFlag deliberately does NOT validate the config
// value (deferred validation per the ticket-3 architectural contract),
// the failure surfaces at the credstore.Open call site. credstore's
// own error text names the config knob ("config keyring.backend") so
// the user has an actionable selector to fix without us double-prefixing.
func TestOpenWith_InvalidConfigBackend_FailsClosed(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("SLACK_CHAT_API_KEYRING_BACKEND", "")
	SetBackendFlagOverride("", false)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	cfg := &appconfig.Config{
		CredentialRef: appconfig.DefaultCredentialRef,
		Keyring:       appconfig.KeyringConfig{Backend: "bogus"},
	}
	_, err := openWith(cfg, false, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, credstore.ErrBackendNotImplemented) {
		t.Errorf("errors.Is(_, ErrBackendNotImplemented) = false; err=%v", err)
	}
	if !strings.Contains(err.Error(), "keyring.backend") {
		t.Errorf("error should name the config knob ('keyring.backend'); got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the bad value; got %q", err.Error())
	}
}

// TestOpenWith_InvalidFlagBackend_AttributesToFlag drives a bogus
// --backend value through openWith via the package override and asserts
// the error is attributed to --backend, not keyring.backend.
func TestOpenWith_InvalidFlagBackend_AttributesToFlag(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("SLACK_CHAT_API_KEYRING_BACKEND", "")
	SetBackendFlagOverride("bogus", true)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	cfg := &appconfig.Config{CredentialRef: appconfig.DefaultCredentialRef}
	_, err := openWith(cfg, false, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, credstore.ErrBackendNotImplemented) {
		t.Errorf("errors.Is(_, ErrBackendNotImplemented) = false; err=%v", err)
	}
	if !strings.HasPrefix(err.Error(), "--"+credstore.BackendFlagName+":") {
		t.Errorf("error should be attributed to --%s; got %q", credstore.BackendFlagName, err.Error())
	}
}
