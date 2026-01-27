package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockUserServer(t *testing.T, users map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/users.info", r.URL.Path)
		uid := r.URL.Query().Get("user")
		name, ok := users[uid]
		if !ok {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":    false,
				"error": "user_not_found",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        uid,
				"name":      name,
				"real_name": name,
				"profile": map[string]interface{}{
					"display_name": name,
				},
			},
		})
	}))
}

func TestUserResolver_Resolve(t *testing.T) {
	server := newMockUserServer(t, map[string]string{
		"U001": "alice",
		"U002": "bob",
	})
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)
	r := NewUserResolver(c)

	assert.Equal(t, "alice", r.Resolve("U001"))
	assert.Equal(t, "bob", r.Resolve("U002"))
}

func TestUserResolver_Cache(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":   "U001",
				"name": "alice",
				"profile": map[string]interface{}{
					"display_name": "alice",
				},
			},
		})
	}))
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)
	r := NewUserResolver(c)

	r.Resolve("U001")
	r.Resolve("U001")
	r.Resolve("U001")

	assert.Equal(t, int32(1), callCount.Load(), "should only call API once due to caching")
}

func TestUserResolver_FallbackOnError(t *testing.T) {
	server := newMockUserServer(t, map[string]string{})
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)
	r := NewUserResolver(c)

	assert.Equal(t, "U999", r.Resolve("U999"))
}

func TestUserResolver_EmptyID(t *testing.T) {
	r := NewUserResolver(nil)
	assert.Equal(t, "", r.Resolve(""))
}

func TestUserResolver_ResolveMentions(t *testing.T) {
	server := newMockUserServer(t, map[string]string{
		"U001": "alice",
		"U002": "bob",
	})
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)
	r := NewUserResolver(c)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no mentions", "hello world", "hello world"},
		{"single mention", "hey <@U001>!", "hey @alice!"},
		{"multiple mentions", "<@U001> and <@U002> are here", "@alice and @bob are here"},
		{"unknown user", "hey <@U999>", "hey @U999"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, r.ResolveMentions(tt.input))
		})
	}
}
