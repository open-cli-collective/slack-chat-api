package emoji

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

func TestRunList_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/emoji.list", r.URL.Path)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"emoji": map[string]string{
				"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
				"shipit":       "https://emoji.slack-edge.com/shipit.png",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_FiltersAliasesByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"emoji": map[string]string{
				"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
				"pp":           "alias:party-parrot",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{includeAliases: false}

	// Should succeed — aliases filtered out silently
	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_IncludesAliasesWhenRequested(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"emoji": map[string]string{
				"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
				"pp":           "alias:party-parrot",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{includeAliases: true}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    true,
			"emoji": map[string]string{},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	err := runList(opts, c)
	require.NoError(t, err)
}

func TestRunList_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    false,
			"error": "missing_scope",
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	err := runList(opts, c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing_scope")
}
