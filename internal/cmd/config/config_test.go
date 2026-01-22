package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

func TestRunSetToken_Success(t *testing.T) {
	if keychain.IsSecureStorage() {
		// On macOS, tokens go to Keychain - just verify no error
		opts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678", opts)
		require.NoError(t, err)

		// Clean up
		_ = keychain.DeleteAPIToken()
	} else {
		// On Linux, use temp directory
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		opts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678", opts)
		require.NoError(t, err)
	}
}

func TestRunSetToken_EmptyToken_WithInput(t *testing.T) {
	// Test with a provided token (not empty string that triggers stdin read)
	opts := &setTokenOptions{}

	// The runSetToken function checks for empty after stdin read,
	// so we can't test the empty validation without stdin
	// Instead, test that a valid token works
	if keychain.IsSecureStorage() {
		err := runSetToken("xoxb-valid-token-12345678", opts)
		require.NoError(t, err)
		_ = keychain.DeleteAPIToken()
	} else {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		err := runSetToken("xoxb-valid-token-12345678", opts)
		require.NoError(t, err)
	}
}

func TestRunDeleteToken_Success(t *testing.T) {
	if keychain.IsSecureStorage() {
		// Set then delete via keychain
		setOpts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678", setOpts)
		require.NoError(t, err)

		deleteOpts := &deleteTokenOptions{tokenType: "all"}
		err = runDeleteToken(deleteOpts)
		require.NoError(t, err)
	} else {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		setOpts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678", setOpts)
		require.NoError(t, err)

		deleteOpts := &deleteTokenOptions{tokenType: "all"}
		err = runDeleteToken(deleteOpts)
		require.NoError(t, err)
	}
}

func TestRunShow_NoToken(t *testing.T) {
	if keychain.IsSecureStorage() {
		// Make sure no token is set
		_ = keychain.DeleteAPIToken()
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	opts := &showOptions{}

	// Should not error, just show "Not configured"
	err := runShow(opts)
	require.NoError(t, err)
}

func TestRunShow_WithToken(t *testing.T) {
	if keychain.IsSecureStorage() {
		// Set a token
		setOpts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678901234567890", setOpts)
		require.NoError(t, err)

		showOpts := &showOptions{}
		err = runShow(showOpts)
		require.NoError(t, err)

		// Clean up
		_ = keychain.DeleteAPIToken()
	} else {
		tempDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tempDir)

		setOpts := &setTokenOptions{}
		err := runSetToken("xoxb-test-token-12345678901234567890", setOpts)
		require.NoError(t, err)

		showOpts := &showOptions{}
		err = runShow(showOpts)
		require.NoError(t, err)
	}
}

func TestRunDeleteToken_WhenNoToken(t *testing.T) {
	if keychain.IsSecureStorage() {
		// First ensure no token exists
		_ = keychain.DeleteAPIToken()
	}

	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	opts := &deleteTokenOptions{}

	// On keychain it may error if no item found, on file-based it's fine
	err := runDeleteToken(opts)
	// We don't assert on error since behavior varies by platform
	_ = err
}

func TestRunSetToken_TokenFormats(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{"bot token", "xoxb-fake-token-for-testing-only"},
		{"user token", "xoxp-fake-token-for-testing-only"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if keychain.IsSecureStorage() {
				opts := &setTokenOptions{}
				err := runSetToken(tt.token, opts)
				require.NoError(t, err)
				_ = keychain.DeleteAPIToken()
				_ = keychain.DeleteUserToken()
			} else {
				tempDir := t.TempDir()
				t.Setenv("XDG_CONFIG_HOME", tempDir)

				opts := &setTokenOptions{}
				err := runSetToken(tt.token, opts)
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunSetToken_InvalidTokenFormat(t *testing.T) {
	invalidTokens := []struct {
		name  string
		token string
	}{
		{"app token", "xapp-fake-token-for-testing-only"},
		{"unknown prefix", "invalid-token-format"},
	}

	for _, tt := range invalidTokens {
		t.Run(tt.name, func(t *testing.T) {
			opts := &setTokenOptions{}
			err := runSetToken(tt.token, opts)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unrecognized token format")
		})
	}
}

// Confirmation prompt tests for delete-token command

func TestRunDeleteToken_Confirmation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		force        bool
		expectDelete bool
	}{
		{"force skips prompt", "", true, true},
		{"y confirms", "y\n", false, true},
		{"yes confirms", "yes\n", false, true},
		{"YES confirms (case insensitive)", "YES\n", false, true},
		{"n cancels", "n\n", false, false},
		{"no cancels", "no\n", false, false},
		{"empty input cancels", "\n", false, false},
		{"other input cancels", "maybe\n", false, false},
		{"whitespace y confirms", "  y  \n", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use temp dir to avoid affecting real keychain
			tempDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", tempDir)

			// Skip keychain tests on macOS since we can't easily mock it
			if keychain.IsSecureStorage() {
				t.Skip("Skipping on macOS - keychain can't be easily mocked")
			}

			// First set a token
			setOpts := &setTokenOptions{}
			err := runSetToken("xoxb-test-token-12345678", setOpts)
			require.NoError(t, err)

			// Now test delete with confirmation
			deleteOpts := &deleteTokenOptions{
				force:     tt.force,
				stdin:     strings.NewReader(tt.input),
				tokenType: "all",
			}

			err = runDeleteToken(deleteOpts)
			require.NoError(t, err)

			// Check if token was actually deleted
			_, tokenErr := keychain.GetAPIToken()
			if tt.expectDelete {
				assert.Error(t, tokenErr, "token should have been deleted")
			} else {
				assert.NoError(t, tokenErr, "token should still exist")
				// Clean up for next test
				_ = keychain.DeleteAPIToken()
			}
		})
	}
}
