package root

import (
	"strings"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

func TestAsUserAndAsBotMutualExclusivity(t *testing.T) {
	originalAsUser, originalAsBot := asUser, asBot
	defer func() {
		asUser, asBot = originalAsUser, originalAsBot
		client.ResetTokenMode()
	}()

	asUser, asBot = true, true
	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err == nil {
		t.Fatal("expected error when both --as-user and --as-bot are set")
	}
	if err.Error() != "cannot use both --as-user and --as-bot flags together" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// resolveErr runs PersistentPreRunE with the given flag state, then
// client.New() against a hermetic empty keyring, returning the resulting
// error text. testutil.Setup forces the file backend in a temp HOME so this
// never touches the real OS keyring (§1.12 hermeticity).
func resolveErr(t *testing.T, user, bot bool) string {
	t.Helper()
	testutil.Setup(t)
	origU, origB := asUser, asBot
	t.Cleanup(func() {
		asUser, asBot = origU, origB
		client.ResetTokenMode()
	})
	asUser, asBot = user, bot
	client.ResetTokenMode()
	if err := rootCmd.PersistentPreRunE(rootCmd, []string{}); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}
	_, err := client.New()
	if err == nil {
		t.Fatal("expected a missing-credential error against an empty keyring")
	}
	return err.Error()
}

func TestAsUserFlagSelectsUserToken(t *testing.T) {
	msg := resolveErr(t, true, false)
	if !strings.Contains(msg, "user token") {
		t.Errorf("--as-user should resolve the user token; got %q", msg)
	}
}

func TestAsBotFlagSelectsBotToken(t *testing.T) {
	msg := resolveErr(t, false, true)
	if !strings.Contains(msg, "bot token") || strings.Contains(msg, "user token") {
		t.Errorf("--as-bot should resolve the bot token; got %q", msg)
	}
}

func TestNoFlagsDefaultsBotMode(t *testing.T) {
	msg := resolveErr(t, false, false)
	if !strings.Contains(msg, "bot token") || strings.Contains(msg, "user token") {
		t.Errorf("default should resolve the bot token; got %q", msg)
	}
}
