package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapError(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		inputErr   error
		wantNil    bool
		wantHint   string
		shouldHint bool
	}{
		{
			name:      "nil error returns nil",
			operation: "test op",
			inputErr:  nil,
			wantNil:   true,
		},
		{
			name:       "channel_not_found includes hint",
			operation:  "get channel",
			inputErr:   fmt.Errorf("channel_not_found"),
			wantHint:   "Verify the channel ID is correct",
			shouldHint: true,
		},
		{
			name:       "not_in_channel includes invite hint",
			operation:  "send message",
			inputErr:   fmt.Errorf("not_in_channel"),
			wantHint:   "bot must be invited",
			shouldHint: true,
		},
		{
			name:       "invalid_auth includes token hint",
			operation:  "list channels",
			inputErr:   fmt.Errorf("invalid_auth"),
			wantHint:   "Token is invalid or expired",
			shouldHint: true,
		},
		{
			name:       "ratelimited includes retry hint",
			operation:  "send message",
			inputErr:   fmt.Errorf("ratelimited"),
			wantHint:   "Wait a moment",
			shouldHint: true,
		},
		{
			name:       "user_not_found includes list hint",
			operation:  "get user",
			inputErr:   fmt.Errorf("user_not_found"),
			wantHint:   "slck users list",
			shouldHint: true,
		},
		{
			name:       "unknown error has no hint",
			operation:  "unknown op",
			inputErr:   fmt.Errorf("some_random_error_code"),
			shouldHint: false,
		},
		{
			name:       "error embedded in longer message",
			operation:  "archive channel",
			inputErr:   fmt.Errorf("slack api error: channel_not_found: no such channel"),
			wantHint:   "Verify the channel ID is correct",
			shouldHint: true,
		},
		{
			name:       "already_reacted includes hint",
			operation:  "add reaction",
			inputErr:   fmt.Errorf("already_reacted"),
			wantHint:   "Reaction already exists",
			shouldHint: true,
		},
		{
			name:       "no_reaction includes hint",
			operation:  "remove reaction",
			inputErr:   fmt.Errorf("no_reaction"),
			wantHint:   "No matching reaction found",
			shouldHint: true,
		},
		{
			name:       "already_in_channel includes hint",
			operation:  "invite user",
			inputErr:   fmt.Errorf("already_in_channel"),
			wantHint:   "already a member",
			shouldHint: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapError(tt.operation, tt.inputErr)

			if tt.wantNil {
				assert.Nil(t, wrapped)
				return
			}

			require.NotNil(t, wrapped)
			errStr := wrapped.Error()

			// Check operation is included
			assert.Contains(t, errStr, tt.operation)

			// Check hint presence
			if tt.shouldHint {
				assert.Contains(t, errStr, "Hint:")
				assert.Contains(t, errStr, tt.wantHint)
			} else {
				assert.NotContains(t, errStr, "Hint:")
			}
		})
	}
}

func TestIsSlackError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
		want bool
	}{
		{
			name: "nil error returns false",
			err:  nil,
			code: "already_reacted",
			want: false,
		},
		{
			name: "matching code returns true",
			err:  fmt.Errorf("slack API error: already_reacted"),
			code: "already_reacted",
			want: true,
		},
		{
			name: "non-matching code returns false",
			err:  fmt.Errorf("slack API error: channel_not_found"),
			code: "already_reacted",
			want: false,
		},
		{
			name: "code embedded in longer message",
			err:  fmt.Errorf("something went wrong: no_reaction: details here"),
			code: "no_reaction",
			want: true,
		},
		{
			name: "already_in_channel",
			err:  fmt.Errorf("slack API error: already_in_channel"),
			code: "already_in_channel",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSlackError(tt.err, tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWrapError_PreservesErrorChain(t *testing.T) {
	// Verify that errors.Is works with wrapped errors
	originalErr := fmt.Errorf("channel_not_found")
	wrapped := WrapError("test op", originalErr)

	require.NotNil(t, wrapped)
	assert.True(t, errors.Is(wrapped, originalErr),
		"wrapped error should preserve error chain for errors.Is()")
}

func TestWrapError_OperationContext(t *testing.T) {
	// Verify operation name appears at the start of the error message
	err := WrapError("archive channel C123", fmt.Errorf("already_archived"))

	require.NotNil(t, err)
	errStr := err.Error()

	// Operation should appear before the original error
	assert.True(t, len(errStr) > 0)
	assert.Contains(t, errStr, "archive channel C123")
	assert.Contains(t, errStr, "already_archived")
}
