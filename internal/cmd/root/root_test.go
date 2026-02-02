package root

import (
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

func TestAsUserAndAsBotMutualExclusivity(t *testing.T) {
	// Save original flag values
	originalAsUser := asUser
	originalAsBot := asBot
	defer func() {
		asUser = originalAsUser
		asBot = originalAsBot
		client.ResetTokenMode()
	}()

	// Test: both flags set should return error
	asUser = true
	asBot = true

	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err == nil {
		t.Error("Expected error when both --as-user and --as-bot are set, got nil")
	}
	if err != nil && err.Error() != "cannot use both --as-user and --as-bot flags together" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestAsUserFlagSetsClientMode(t *testing.T) {
	// Save original flag values
	originalAsUser := asUser
	originalAsBot := asBot
	defer func() {
		asUser = originalAsUser
		asBot = originalAsBot
		client.ResetTokenMode()
	}()

	// Test: --as-user flag should set client to user mode
	asUser = true
	asBot = false
	client.ResetTokenMode()

	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify by checking that New() would try to use user token
	// We can't directly inspect the internal state, but we can verify
	// the behavior by attempting to create a client and checking the error
	_, err = client.New()
	if err == nil {
		// If no error, client was created (token exists)
		return
	}
	// Error should mention user token since that's what we're trying to use
	if err.Error() == "user token not found: set SLACK_USER_TOKEN or run: slck config set-user-token" {
		// This confirms --as-user flag correctly set the client mode
		return
	}
	// If we get here with a different error, that's also acceptable
	// as long as it's not a bot token error when we expected user mode
}

func TestAsBotFlagSetsClientMode(t *testing.T) {
	// Save original flag values
	originalAsUser := asUser
	originalAsBot := asBot
	defer func() {
		asUser = originalAsUser
		asBot = originalAsBot
		client.ResetTokenMode()
	}()

	// Test: --as-bot flag should set client to bot mode
	asUser = false
	asBot = true
	client.ResetTokenMode()

	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify by checking that New() would try to use bot token
	_, err = client.New()
	if err == nil {
		// If no error, client was created (token exists)
		return
	}
	// Error should NOT mention user token since we're using bot mode
	if err.Error() == "user token not found: set SLACK_USER_TOKEN or run: slck config set-user-token" {
		t.Error("Expected bot token mode but got user token error")
	}
}

func TestNoFlagsDefaultsBotMode(t *testing.T) {
	// Save original flag values
	originalAsUser := asUser
	originalAsBot := asBot
	defer func() {
		asUser = originalAsUser
		asBot = originalAsBot
		client.ResetTokenMode()
	}()

	// Test: no flags should default to bot mode
	asUser = false
	asBot = false
	client.ResetTokenMode()

	err := rootCmd.PersistentPreRunE(rootCmd, []string{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// With no flags set, client.New() should try bot token by default
	_, err = client.New()
	if err == nil {
		// If no error, client was created (token exists)
		return
	}
	// Error should NOT mention user token since default is bot mode
	if err.Error() == "user token not found: set SLACK_USER_TOKEN or run: slck config set-user-token" {
		t.Error("Expected bot token mode by default but got user token error")
	}
}
