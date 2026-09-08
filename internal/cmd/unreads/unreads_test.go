package unreads

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func TestRunListGroupsUnreadConversations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response any
		switch r.URL.Path {
		case "/users.conversations":
			assert.Equal(t, "public_channel,private_channel,im,mpim", r.URL.Query().Get("types"))
			response = map[string]any{
				"ok": true,
				"channels": []map[string]any{
					{"id": "C1", "name": "alerts"},
					{"id": "C2", "name": "general"},
					{"id": "D1", "is_im": true, "user": "U1", "priority": 0.2},
					{"id": "D2", "is_im": true, "user": "B1", "priority": 0.1},
					{"id": "D3", "is_im": true, "user": "U1", "priority": 0},
					{"id": "G1", "name": "project", "is_mpim": true, "priority": 0.05},
				},
			}
		case "/users.list":
			response = map[string]any{
				"ok": true,
				"members": []map[string]any{
					{"id": "U1", "name": "alice", "profile": map[string]any{"display_name": "Alice"}},
					{"id": "B1", "name": "helper", "is_app_user": true},
				},
			}
		case "/conversations.info":
			id := r.URL.Query().Get("channel")
			assert.NotEqual(t, "D3", id, "dormant DMs should not be scanned")
			channel := map[string]any{"id": id, "last_read": "1.000000"}
			switch id {
			case "D1":
				channel["unread_count"] = 2
			case "D2":
				channel["unread_count"] = 1
			case "G1":
				channel["last_read"] = "0"
			}
			response = map[string]any{"ok": true, "channel": channel}
		case "/conversations.history":
			assert.Equal(t, "1", r.URL.Query().Get("limit"))
			if r.URL.Query().Get("channel") == "G1" {
				assert.Empty(t, r.URL.Query().Get("oldest"))
			} else {
				assert.Equal(t, "1.000000", r.URL.Query().Get("oldest"))
			}
			messages := []map[string]any{}
			if id := r.URL.Query().Get("channel"); id == "C1" || id == "G1" {
				messages = append(messages, map[string]any{"ts": "2.000000", "text": "new"})
			}
			response = map[string]any{"ok": true, "messages": messages}
		default:
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "xoxp-test", nil)
	originalWriter := output.Writer
	defer func() { output.Writer = originalWriter }()

	var buf bytes.Buffer
	output.Writer = &buf
	require.NoError(t, runList(&listOptions{}, c))
	assert.Contains(t, buf.String(), "Channels")
	assert.Contains(t, buf.String(), "C1")
	assert.NotContains(t, buf.String(), "C2")
	assert.Contains(t, buf.String(), "Direct messages")
	assert.Contains(t, buf.String(), "Alice")
	assert.Contains(t, buf.String(), "project")
	assert.NotContains(t, buf.String(), "Agents & apps")
	assert.NotContains(t, buf.String(), "helper")

	buf.Reset()
	require.NoError(t, runList(&listOptions{includeApps: true}, c))
	assert.Contains(t, buf.String(), "Agents & apps")
	assert.Contains(t, buf.String(), "D2")
	assert.Contains(t, buf.String(), "helper")
}
