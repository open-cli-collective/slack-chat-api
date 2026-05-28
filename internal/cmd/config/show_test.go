package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

// TestRunShow_ReportsKeyringBackendSelector seeds keyring.backend in
// config.yml and asserts both text and JSON output surface the selector
// — proves the showStatus.KeyringBackend wiring is reachable through
// runShow.
func TestRunShow_ReportsKeyringBackendSelector(t *testing.T) {
	testutil.Setup(t)
	cfg, err := appconfig.LoadForRuntime()
	require.NoError(t, err)
	cfg.Keyring.Backend = "file"
	require.NoError(t, cfg.Save())

	// Text output
	out, err := captureOutput(t, func() error { return runShow(&showOptions{}) })
	require.NoError(t, err)
	if !strings.Contains(out, "keyring.backend: file (config.yml)") {
		t.Errorf("text show missing selector line:\n%s", out)
	}

	// JSON output via the local --json carve-out flag (post-#173 — the
	// global -o json is gone; --json is the only JSON surface).
	jsonOut, err := captureOutput(t, func() error { return runShow(&showOptions{json: true}) })
	require.NoError(t, err)
	var st showStatus
	if err := json.Unmarshal([]byte(jsonOut), &st); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, jsonOut)
	}
	if st.KeyringBackend != "file" {
		t.Errorf("KeyringBackend = %q, want %q", st.KeyringBackend, "file")
	}
}

// TestRunShow_OmitsKeyringBackendWhenUnset asserts the omitempty path:
// no config.yml selector → no `keyring.backend:` line in text, no
// `keyring_backend` key in JSON.
func TestRunShow_OmitsKeyringBackendWhenUnset(t *testing.T) {
	testutil.Setup(t)
	// Default config has Keyring.Backend == "".

	out, err := captureOutput(t, func() error { return runShow(&showOptions{}) })
	require.NoError(t, err)
	if strings.Contains(out, "keyring.backend:") {
		t.Errorf("text show emitted selector line when unset:\n%s", out)
	}

	jsonOut, err := captureOutput(t, func() error { return runShow(&showOptions{json: true}) })
	require.NoError(t, err)
	if strings.Contains(jsonOut, `"keyring_backend"`) {
		t.Errorf("json show emitted keyring_backend when unset: %s", jsonOut)
	}
}

// TestRunShow_JSONFlagOverridesGlobalOutput pins the documented
// composition rule: when --json is set, the local carve-out takes
// precedence over -o text/table. Sets output.OutputFormat to FormatTable
// (the state root's PersistentPreRunE would leave for `slck config show
// --json -o table`) and asserts runShow still emits a JSON envelope.
func TestRunShow_JSONFlagOverridesGlobalOutput(t *testing.T) {
	testutil.Setup(t)
	priorFormat := output.OutputFormat
	output.OutputFormat = output.FormatTable
	t.Cleanup(func() { output.OutputFormat = priorFormat })

	out, err := captureOutput(t, func() error { return runShow(&showOptions{json: true}) })
	require.NoError(t, err)
	var st showStatus
	if err := json.Unmarshal([]byte(out), &st); err != nil {
		t.Fatalf("--json output is not valid JSON despite -o table: %v\n%s", err, out)
	}
	if strings.Contains(out, "Credential ref:") || strings.Contains(out, "Backend:") {
		t.Fatalf("--json leaked human-readable lines: %s", out)
	}
}
