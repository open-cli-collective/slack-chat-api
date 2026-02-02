package client

import (
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

func TestNew(t *testing.T) {
	// Save and restore environment and token pointer
	originalEnv := os.Getenv("SLCK_AS_USER")
	originalPtr := useUserToken
	defer func() {
		if originalEnv != "" {
			os.Setenv("SLCK_AS_USER", originalEnv)
		} else {
			os.Unsetenv("SLCK_AS_USER")
		}
		useUserToken = originalPtr
	}()

	// Check if we have tokens available for testing
	hasAPIToken := keychain.HasStoredToken() || os.Getenv("SLACK_API_TOKEN") != ""
	hasUserToken := keychain.HasStoredUserToken() || os.Getenv("SLACK_USER_TOKEN") != ""

	trueVal := true
	falseVal := false

	tests := []struct {
		name        string
		tokenPtr    *bool
		envValue    string
		expectUser  bool
		description string
	}{
		{
			name:        "explicit user token",
			tokenPtr:    &trueVal,
			envValue:    "",
			expectUser:  true,
			description: "When explicitly set to user, should use user token",
		},
		{
			name:        "explicit bot token",
			tokenPtr:    &falseVal,
			envValue:    "",
			expectUser:  false,
			description: "When explicitly set to bot, should use bot token",
		},
		{
			name:        "unset with no env uses bot token",
			tokenPtr:    nil,
			envValue:    "",
			expectUser:  false,
			description: "When unset and no env var, should use bot token",
		},
		{
			name:        "unset with env true uses user token",
			tokenPtr:    nil,
			envValue:    "true",
			expectUser:  true,
			description: "When unset and SLCK_AS_USER=true, should use user token",
		},
		{
			name:        "unset with env 1 uses user token",
			tokenPtr:    nil,
			envValue:    "1",
			expectUser:  true,
			description: "When unset and SLCK_AS_USER=1, should use user token",
		},
		{
			name:        "unset with env false uses bot token",
			tokenPtr:    nil,
			envValue:    "false",
			expectUser:  false,
			description: "When unset and SLCK_AS_USER=false, should use bot token",
		},
		{
			name:        "explicit user ignores env var",
			tokenPtr:    &trueVal,
			envValue:    "false",
			expectUser:  true,
			description: "Explicit user should ignore env var",
		},
		{
			name:        "explicit bot overrides env var",
			tokenPtr:    &falseVal,
			envValue:    "true",
			expectUser:  false,
			description: "Explicit bot should override SLCK_AS_USER=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set token pointer and environment
			useUserToken = tt.tokenPtr
			if tt.envValue != "" {
				os.Setenv("SLCK_AS_USER", tt.envValue)
			} else {
				os.Unsetenv("SLCK_AS_USER")
			}

			// Call the function - may succeed if tokens are present, or fail if not
			client, err := New()

			// Case 1: If we have the appropriate token, client should be created successfully
			if tt.expectUser && hasUserToken {
				if err != nil {
					t.Errorf("Expected success with user token, got error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client to be created")
				}
				return
			}

			if !tt.expectUser && hasAPIToken {
				if err != nil {
					t.Errorf("Expected success with bot token, got error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected client to be created")
				}
				return
			}

			// Case 2: If we don't have the appropriate token, should get specific error
			if err == nil {
				t.Errorf("Expected error due to missing token, got nil")
				return
			}

			// Check error message indicates correct token type was attempted
			errMsg := err.Error()
			if tt.expectUser {
				// Should try to get user token
				if !strings.Contains(errMsg, "user token") && !strings.Contains(errMsg, "xoxp-") {
					t.Errorf("Expected user token error, got: %s", errMsg)
				}
			} else {
				// Should try to get bot token (but NOT specifically request user token)
				if strings.Contains(errMsg, "user token") || strings.Contains(errMsg, "xoxp-") {
					t.Errorf("Expected bot token error, got user token error: %s", errMsg)
				}
			}
		})
	}
}

func TestSetAsUser(t *testing.T) {
	// Save original value
	original := useUserToken
	defer func() {
		useUserToken = original
	}()

	SetAsUser(true)
	if useUserToken == nil || !*useUserToken {
		t.Errorf("Expected useUserToken to point to true after SetAsUser(true)")
	}

	SetAsUser(false)
	if useUserToken == nil || *useUserToken {
		t.Errorf("Expected useUserToken to point to false after SetAsUser(false)")
	}
}

func TestResetTokenMode(t *testing.T) {
	// Set to user mode
	SetAsUser(true)
	if useUserToken == nil {
		t.Errorf("Expected useUserToken to be non-nil after SetAsUser(true)")
	}

	// Reset
	ResetTokenMode()
	if useUserToken != nil {
		t.Errorf("Expected useUserToken to be nil after reset, got %v", useUserToken)
	}
}
