package keychain

import (
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
