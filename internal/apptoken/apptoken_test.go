package apptoken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	access  = "xoxe.xoxp-1-access-canary-3f9a"
	refresh = "xoxe-1-refresh-canary-81c2"
)

func TestParseAcceptsTheIngressContract(t *testing.T) {
	p, err := Parse(`{"token": "` + access + `", "refresh_token": "` + refresh + `", "exp": 1790043200}`)
	require.NoError(t, err)
	assert.Equal(t, access, p.Token)
	assert.Equal(t, refresh, p.RefreshToken)
	assert.Equal(t, time.Unix(1790043200, 0), p.Expiry())
}

func TestParseRoundTripsAStoredPair(t *testing.T) {
	stored, err := Pair{Token: access, RefreshToken: refresh, Exp: 1790043200, TeamID: "T0TEST", UserID: "U0TEST"}.Encode()
	require.NoError(t, err)
	p, err := Parse(stored)
	require.NoError(t, err)
	assert.Equal(t, "T0TEST", p.TeamID)
}

func TestParseRejectionsNeverIncludeTheValue(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":       access,
		"unknown field":  `{"token": "` + access + `", "refresh_token": "` + refresh + `", "exp": 1, "access": "x"}`,
		"bot token":      `{"token": "xoxb-1-canary", "refresh_token": "` + refresh + `", "exp": 1}`,
		"swapped tokens": `{"token": "` + refresh + `", "refresh_token": "` + access + `", "exp": 1}`,
		"missing expiry": `{"token": "` + access + `", "refresh_token": "` + refresh + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(raw)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "canary")
		})
	}
}

func TestNeedsRotation(t *testing.T) {
	exp := time.Unix(1790043200, 0)
	p := Pair{Exp: exp.Unix()}
	assert.False(t, p.NeedsRotation(exp.Add(-time.Hour)))
	assert.True(t, p.NeedsRotation(exp.Add(-RotateWithin)))
	assert.True(t, p.NeedsRotation(exp.Add(time.Hour)))
}
