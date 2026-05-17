// Package noleak is the §1.12 acceptance "no-leak" suite. It loads a
// known-distinctive token into a hermetic keyring, drives every
// credential-surface command class through its real cobra command, captures
// EVERY output channel (os.Stdout, os.Stderr, and output.Writer), and fails
// if the token — or any prefix of it — appears anywhere. This is the test
// obligation B2/B3 reuse verbatim.
package noleak

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/cli-common/credstore"

	cfgcmd "github.com/open-cli-collective/slack-chat-api/internal/cmd/config"
	initcmd "github.com/open-cli-collective/slack-chat-api/internal/cmd/initcmd"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/setcred"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/whoami"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

const secret = "xoxb-NOLEAK-canary-7f3a9c2e1d8b4a6f0011"

// captureAll redirects os.Stdout, os.Stderr and output.Writer, runs fn, and
// returns everything written to any of them.
func captureAll(t *testing.T, stdin string, fn func()) string {
	t.Helper()
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	origOut, origErr, origW := os.Stdout, os.Stderr, output.Writer
	os.Stdout, os.Stderr = wOut, wErr
	var ob bytes.Buffer
	output.Writer = &ob

	var inR *os.File
	if stdin != "" {
		var inW *os.File
		inR, inW, _ = os.Pipe()
		origIn := os.Stdin
		os.Stdin = inR
		go func() { _, _ = inW.WriteString(stdin); _ = inW.Close() }()
		t.Cleanup(func() { os.Stdin = origIn })
	}

	done := make(chan string, 2)
	go func() { b, _ := io.ReadAll(rOut); done <- string(b) }()
	go func() { b, _ := io.ReadAll(rErr); done <- string(b) }()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr, output.Writer = origOut, origErr, origW
	out1, out2 := <-done, <-done
	return out1 + out2 + ob.String()
}

func seed(t *testing.T) {
	t.Helper()
	st, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, st.SetBotToken(secret))
	require.NoError(t, st.Close())
}

func assertNoLeak(t *testing.T, name, captured string) {
	t.Helper()
	if leak := credstore.NoLeakAssertion([]byte(captured), secret); leak != nil {
		t.Fatalf("%s leaked the secret (%v).\n--- captured ---\n%s", name, leak, captured)
	}
}

func TestNoLeak_ConfigShowText(t *testing.T) {
	testutil.Setup(t)
	seed(t)
	c := cfgcmd.NewCmd()
	c.SetArgs([]string{"show"})
	out := captureAll(t, "", func() { _ = c.Execute() })
	require.Contains(t, out, "present")
	assertNoLeak(t, "config show", out)
}

func TestNoLeak_ConfigShowJSON(t *testing.T) {
	testutil.Setup(t)
	seed(t)
	output.JSON = true
	t.Cleanup(func() { output.JSON = false })
	c := cfgcmd.NewCmd()
	c.SetArgs([]string{"show"})
	assertNoLeak(t, "config show -o json", captureAll(t, "", func() { _ = c.Execute() }))
}

func TestNoLeak_DeleteTokenAndClear(t *testing.T) {
	for _, args := range [][]string{{"delete-token", "--force"}, {"clear"}, {"clear", "--all"}} {
		testutil.Setup(t)
		seed(t)
		c := cfgcmd.NewCmd()
		c.SetArgs(args)
		assertNoLeak(t, "config "+strings.Join(args, " "),
			captureAll(t, "", func() { _ = c.Execute() }))
	}
}

func TestNoLeak_SetCredentialStdin(t *testing.T) {
	testutil.Setup(t)
	c := setcred.NewCmd()
	c.SetArgs([]string{"--key", "bot_token", "--stdin"})
	out := captureAll(t, secret+"\n", func() { _ = c.Execute() })
	assertNoLeak(t, "set-credential --stdin", out)
}

func TestNoLeak_InitFromEnv(t *testing.T) {
	testutil.Setup(t)
	t.Setenv("NOLEAK_BOT", secret)
	c := initcmd.NewCmd()
	c.SetArgs([]string{"--bot-token-from-env", "NOLEAK_BOT", "--no-verify"})
	out := captureAll(t, "", func() { _ = c.Execute() })
	assertNoLeak(t, "init --bot-token-from-env", out)
}

func TestNoLeak_Whoami(t *testing.T) {
	testutil.Setup(t)
	seed(t)
	c := whoami.NewCmd()
	c.SetArgs([]string{})
	// whoami will fail to reach Slack (hermetic env); we only care that no
	// output channel echoes the seeded token.
	assertNoLeak(t, "whoami", captureAll(t, "", func() { _ = c.Execute() }))
}
