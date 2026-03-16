package emoji

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func captureOutput(fn func()) string {
	var buf bytes.Buffer
	output.Writer = &buf
	defer func() { output.Writer = os.Stdout }()
	fn()
	return buf.String()
}

func emojiServer(emojis map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":    true,
			"emoji": emojis,
		})
	}))
}

func TestRunList_Success(t *testing.T) {
	server := emojiServer(map[string]string{
		"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
		"shipit":       "https://emoji.slack-edge.com/shipit.png",
	})
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	out := captureOutput(func() {
		err := runList(opts, c)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "party-parrot")
	assert.Contains(t, out, "shipit")
}

func TestRunList_OutputIsSorted(t *testing.T) {
	server := emojiServer(map[string]string{
		"zebra":    "https://emoji.slack-edge.com/zebra.png",
		"alpaca":   "https://emoji.slack-edge.com/alpaca.png",
		"mushroom": "https://emoji.slack-edge.com/mushroom.png",
	})
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	out := captureOutput(func() {
		err := runList(opts, c)
		require.NoError(t, err)
	})

	assert.Equal(t, "alpaca\nmushroom\nzebra\n", out)
}

func TestRunList_FiltersAliasesByDefault(t *testing.T) {
	server := emojiServer(map[string]string{
		"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
		"pp":           "alias:party-parrot",
		"shipit":       "https://emoji.slack-edge.com/shipit.png",
	})
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{includeAliases: false}

	out := captureOutput(func() {
		err := runList(opts, c)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "party-parrot")
	assert.Contains(t, out, "shipit")
	assert.NotContains(t, out, "pp")
}

func TestRunList_IncludesAliasesWhenRequested(t *testing.T) {
	server := emojiServer(map[string]string{
		"party-parrot": "https://emoji.slack-edge.com/party-parrot.gif",
		"pp":           "alias:party-parrot",
	})
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{includeAliases: true}

	out := captureOutput(func() {
		err := runList(opts, c)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "party-parrot")
	assert.Contains(t, out, "pp")
}

func TestRunList_Empty(t *testing.T) {
	server := emojiServer(map[string]string{})
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &listOptions{}

	out := captureOutput(func() {
		err := runList(opts, c)
		require.NoError(t, err)
	})

	assert.Contains(t, out, "No custom emoji found")
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
