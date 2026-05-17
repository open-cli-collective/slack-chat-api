package config

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

// TestNoLeak_CredentialSurface is the §1.12 acceptance "no-leak" test for
// slck's credential-surface commands: load a known-distinctive token into
// the keyring, run every credential command class, capture stdout+stderr,
// and fail if the token (or any prefix of it) appears. The API commands
// never echo the token (it travels only in the Authorization header), so
// the leak surface is exactly these commands.
func TestNoLeak_CredentialSurface(t *testing.T) {
	const secret = "xoxb-NOLEAK-canary-7f3a9c2e1d8b4a6f"

	run := func(name string, fn func() error) string {
		t.Helper()
		out, err := captureOutput(t, fn)
		// JSON mode toggles a global; reset so it can't bleed.
		t.Cleanup(func() { output.JSON = false })
		require.NoError(t, err, name)
		if leak := credstore.NoLeakAssertion([]byte(out), secret); leak != nil {
			t.Fatalf("%s leaked the secret: %v\noutput:\n%s", name, leak, out)
		}
		return out
	}

	seed := func() {
		st, err := keychain.Open()
		require.NoError(t, err)
		require.NoError(t, st.SetBotToken(secret))
		require.NoError(t, st.Close())
	}

	t.Run("show text", func(t *testing.T) {
		testutil.Setup(t)
		seed()
		out := run("config show", func() error { return runShow(&showOptions{}) })
		require.Contains(t, out, "present")
	})

	t.Run("show json", func(t *testing.T) {
		testutil.Setup(t)
		seed()
		output.JSON = true
		run("config show -o json", func() error { return runShow(&showOptions{}) })
	})

	t.Run("delete-token", func(t *testing.T) {
		testutil.Setup(t)
		seed()
		run("config delete-token", func() error {
			return runDeleteToken(&deleteTokenOptions{force: true, tokenType: "all"})
		})
	})

	t.Run("clear", func(t *testing.T) {
		testutil.Setup(t)
		seed()
		run("config clear", func() error { return runClear(&clearOptions{}) })
	})

	t.Run("clear --all", func(t *testing.T) {
		testutil.Setup(t)
		seed()
		run("config clear --all", func() error { return runClear(&clearOptions{all: true}) })
	})
}
