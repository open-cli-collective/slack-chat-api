package workspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/piekstra/slack-chat-api/internal/client"
)

func TestRunInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/team.info", r.URL.Path)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"team": map[string]interface{}{
				"id":     "T123456",
				"name":   "My Workspace",
				"domain": "myworkspace",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &infoOptions{}

	err := runInfo(opts, c)
	require.NoError(t, err)
}

func TestRunInfo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "invalid_auth",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &infoOptions{}

	err := runInfo(opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_auth")
}
