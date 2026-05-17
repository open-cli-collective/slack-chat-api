package setcred

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

func capture(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	orig := output.Writer
	output.Writer = &buf
	t.Cleanup(func() { output.Writer = orig })
	return buf.String(), fn()
}

func TestSetCredential_Stdin(t *testing.T) {
	testutil.Setup(t)
	const secret = "xoxb-SETCRED-stdin-canary-12345"
	out, err := capture(t, func() error {
		return run(&options{key: keychain.KeyBotToken, stdin: true,
			in: strings.NewReader(secret + "\n")})
	})
	require.NoError(t, err)
	if leak := credstore.NoLeakAssertion([]byte(out), secret); leak != nil {
		t.Fatalf("set-credential echoed the value: %v", leak)
	}
	st, err := keychain.Open()
	require.NoError(t, err)
	defer func() { _ = st.Close() }()
	got, err := st.BotToken()
	require.NoError(t, err)
	require.Equal(t, secret, got)
}

func TestSetCredential_FromEnv(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("MY_SLACK_SECRET", "xoxp-fromenv-value")
	_, err := capture(t, func() error {
		return run(&options{key: keychain.KeyUserToken, fromEnv: "MY_SLACK_SECRET"})
	})
	require.NoError(t, err)
	st, _ := keychain.Open()
	defer func() { _ = st.Close() }()
	v, err := st.UserToken()
	require.NoError(t, err)
	require.Equal(t, "xoxp-fromenv-value", v)
}

func TestSetCredential_RejectsDisallowedKey(t *testing.T) {
	testutil.Setup(t)
	_, err := capture(t, func() error {
		return run(&options{key: "totally_bogus", stdin: true, in: strings.NewReader("x")})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestSetCredential_RequiresExactlyOneSource(t *testing.T) {
	testutil.Setup(t)
	// neither
	_, err := capture(t, func() error { return run(&options{key: keychain.KeyBotToken}) })
	require.Error(t, err)
	// both
	_, err = capture(t, func() error {
		return run(&options{key: keychain.KeyBotToken, stdin: true, fromEnv: "X"})
	})
	require.Error(t, err)
}

func TestSetCredential_EmptyRejected(t *testing.T) {
	testutil.Setup(t)
	_, err := capture(t, func() error {
		return run(&options{key: keychain.KeyBotToken, stdin: true, in: strings.NewReader("\n")})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty")
}
