package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

const distinctiveTok = "xoxb-DISTINCTIVE-secret-value-9999"

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	orig := output.Writer
	output.Writer = &buf
	t.Cleanup(func() { output.Writer = orig })
	err := fn()
	return buf.String(), err
}

// set-token is hard-deprecated (§2.4): every invocation fails and points at
// set-credential, accepting the value via no path.
func TestSetTokenHardDeprecated(t *testing.T) {
	cmd := newSetTokenCmd()
	for _, args := range [][]string{{}, {"xoxb-leak-attempt"}} {
		cmd.SetArgs(args)
		err := cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set-credential")
		assert.Contains(t, err.Error(), "removed")
	}
}

func TestRunShow_NoToken(t *testing.T) {
	testutil.Setup(t)
	out, err := captureOutput(t, func() error { return runShow(&showOptions{}) })
	require.NoError(t, err)
	assert.Contains(t, out, "not configured")
}

func TestRunShow_WithToken_NeverPrintsValue(t *testing.T) {
	testutil.Setup(t)
	st, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, st.SetBotToken(distinctiveTok))
	_ = st.Close()

	out, err := captureOutput(t, func() error { return runShow(&showOptions{}) })
	require.NoError(t, err)
	assert.Contains(t, out, "present")
	// §1.11 item 3 / §1.12: not even a masked prefix of the value.
	if leak := credstore.NoLeakAssertion([]byte(out), distinctiveTok); leak != nil {
		t.Fatalf("config show leaked: %v", leak)
	}
}

func TestRunDeleteToken_Confirmation(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		force        bool
		expectDelete bool
	}{
		{"force skips prompt", "", true, true},
		{"y confirms", "y\n", false, true},
		{"YES confirms", "YES\n", false, true},
		{"n cancels", "n\n", false, false},
		{"empty cancels", "\n", false, false},
		{"whitespace y", "  y  \n", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Setup(t)
			seed, err := keychain.Open()
			require.NoError(t, err)
			require.NoError(t, seed.SetBotToken("xoxb-test-token-12345678"))
			_ = seed.Close()

			_, err = captureOutput(t, func() error {
				return runDeleteToken(&deleteTokenOptions{
					force: tc.force, stdin: strings.NewReader(tc.input), tokenType: "all",
				})
			})
			require.NoError(t, err)

			chk, err := keychain.Open()
			require.NoError(t, err)
			defer func() { _ = chk.Close() }()
			assert.Equal(t, !tc.expectDelete, chk.HasBotToken())
		})
	}
}

func TestRunDeleteToken_WhenNoToken(t *testing.T) {
	testutil.Setup(t)
	_, err := captureOutput(t, func() error {
		return runDeleteToken(&deleteTokenOptions{tokenType: "all"})
	})
	require.NoError(t, err)
}

func TestRunClear_Idempotent(t *testing.T) {
	testutil.Setup(t)
	seed, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, seed.SetBotToken("xoxb-test-token-12345678"))
	_ = seed.Close()

	_, err = captureOutput(t, func() error { return runClear(&clearOptions{}) })
	require.NoError(t, err)
	// Second clear is a no-op, still succeeds (§1.7 idempotent).
	_, err = captureOutput(t, func() error { return runClear(&clearOptions{}) })
	require.NoError(t, err)
}
