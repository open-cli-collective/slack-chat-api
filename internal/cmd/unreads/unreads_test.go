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
					{"id": "D1", "is_im": true, "user": "U1", "priority": 0},
					{"id": "D2", "is_im": true, "user": "B1"},
					{"id": "D3", "is_im": true, "user": "U1"},
					{"id": "G1", "name": "project", "is_mpim": true, "priority": 0},
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
	assert.Contains(t, buf.String(), "D1")
	assert.Contains(t, buf.String(), "Alice")
	assert.Contains(t, buf.String(), "G1")
	assert.Contains(t, buf.String(), "project")
	assert.NotContains(t, buf.String(), "Agents & apps")
	assert.NotContains(t, buf.String(), "helper")

	buf.Reset()
	require.NoError(t, runList(&listOptions{includeApps: true}, c))
	assert.Contains(t, buf.String(), "Agents & apps")
	assert.Contains(t, buf.String(), "D2")
	assert.Contains(t, buf.String(), "helper")
}

func TestRunListExclusions(t *testing.T) {
	tests := []struct {
		name        string
		opts        listOptions
		wantScanned []string
		wantOutput  []string
		notOutput   []string
	}{
		{
			name:        "channels",
			opts:        listOptions{excludeChannels: true},
			wantScanned: []string{"D1", "G1"},
			wantOutput:  []string{"D1", "G1"},
			notOutput:   []string{"C1", "D2"},
		},
		{
			name:        "direct messages",
			opts:        listOptions{excludeDMs: true},
			wantScanned: []string{"C1"},
			wantOutput:  []string{"C1"},
			notOutput:   []string{"D1", "D2", "G1"},
		},
		{
			name:        "direct messages with apps included",
			opts:        listOptions{excludeDMs: true, includeApps: true},
			wantScanned: []string{"C1", "D2"},
			wantOutput:  []string{"C1", "D2"},
			notOutput:   []string{"D1", "G1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var scanned []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var response any
				switch r.URL.Path {
				case "/users.conversations":
					response = map[string]any{
						"ok": true,
						"channels": []map[string]any{
							{"id": "C1", "name": "channel"},
							{"id": "D1", "is_im": true, "user": "U1"},
							{"id": "D2", "is_im": true, "user": "B1"},
							{"id": "G1", "name": "group", "is_mpim": true},
						},
					}
				case "/users.list":
					response = map[string]any{
						"ok": true,
						"members": []map[string]any{
							{"id": "U1", "name": "human"},
							{"id": "B1", "name": "app", "is_app_user": true},
						},
					}
				case "/conversations.info":
					id := r.URL.Query().Get("channel")
					scanned = append(scanned, id)
					response = map[string]any{
						"ok":      true,
						"channel": map[string]any{"id": id, "unread_count": 1},
					}
				default:
					t.Fatalf("unexpected request: %s", r.URL.String())
				}
				assert.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer server.Close()

			originalWriter := output.Writer
			defer func() { output.Writer = originalWriter }()
			var buf bytes.Buffer
			output.Writer = &buf

			require.NoError(t, runList(&tt.opts, client.NewWithConfig(server.URL, "xoxp-test", nil)))
			assert.ElementsMatch(t, tt.wantScanned, scanned)
			for _, value := range tt.wantOutput {
				assert.Contains(t, buf.String(), value)
			}
			for _, value := range tt.notOutput {
				assert.NotContains(t, buf.String(), value)
			}
		})
	}
}
