package client

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

const (
	flagBotTok  = "xoxb-flagtest-bot"
	flagUserTok = "xoxp-flagtest-user"
)

// TestNew exercises the --as-bot/--as-user flag and SLCK_AS_USER precedence
// in New(). Tokens are seeded into a hermetic file-backed keyring so every
// branch resolves and we can assert which token New() selected.
func TestNew(t *testing.T) {
	testutil.Setup(t)
	st, err := keychain.Open()
	require.NoError(t, err)
	require.NoError(t, st.SetBotToken(flagBotTok))
	require.NoError(t, st.SetUserToken(flagUserTok))
	require.NoError(t, st.Close())

	trueVal, falseVal := true, false
	cases := []struct {
		name      string
		ptr       *bool
		env       string
		wantToken string
	}{
		{"explicit user", &trueVal, "", flagUserTok},
		{"explicit bot", &falseVal, "", flagBotTok},
		{"unset, no env -> bot", nil, "", flagBotTok},
		{"unset, env true -> user", nil, "true", flagUserTok},
		{"unset, env 1 -> user", nil, "1", flagUserTok},
		{"unset, env false -> bot", nil, "false", flagBotTok},
		{"explicit user ignores env", &trueVal, "false", flagUserTok},
		{"explicit bot overrides env", &falseVal, "true", flagBotTok},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			useUserToken = tc.ptr
			t.Cleanup(func() { useUserToken = nil })
			if tc.env != "" {
				t.Setenv("SLCK_AS_USER", tc.env)
			} else {
				t.Setenv("SLCK_AS_USER", "")
			}

			c, err := New()
			require.NoError(t, err)
			require.NotNil(t, c)
			if c.token != tc.wantToken {
				t.Fatalf("New() selected token %q, want %q", c.token, tc.wantToken)
			}
		})
	}
}

func TestSetAsUser(t *testing.T) {
	original := useUserToken
	defer func() { useUserToken = original }()

	SetAsUser(true)
	if useUserToken == nil || !*useUserToken {
		t.Errorf("Expected useUserToken true after SetAsUser(true)")
	}
	SetAsUser(false)
	if useUserToken == nil || *useUserToken {
		t.Errorf("Expected useUserToken false after SetAsUser(false)")
	}
}

func TestResetTokenMode(t *testing.T) {
	SetAsUser(true)
	if useUserToken == nil {
		t.Errorf("Expected useUserToken non-nil after SetAsUser(true)")
	}
	ResetTokenMode()
	if useUserToken != nil {
		t.Errorf("Expected useUserToken nil after reset, got %v", useUserToken)
	}
}
