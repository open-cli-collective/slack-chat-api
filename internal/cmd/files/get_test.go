package files

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*client.Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	return client.NewWithConfig(server.URL, "test-token", nil), server
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf strings.Builder
	orig := output.Writer
	output.Writer = &buf
	defer func() { output.Writer = orig }()
	fn()
	return buf.String()
}

func TestRunGet_Success(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/files.info", r.URL.Path)
		assert.Equal(t, "F0ABC123", r.URL.Query().Get("file"))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":       "F0ABC123",
				"name":     "report.pdf",
				"title":    "Quarterly report",
				"filetype": "pdf",
				"size":     60620,
			},
		})
	})
	defer server.Close()

	out := captureOutput(t, func() {
		require.NoError(t, runGet("F0ABC123", c))
	})

	assert.Contains(t, out, "ID: F0ABC123\n")
	assert.Contains(t, out, "Name: report.pdf\n")
	assert.Contains(t, out, "Title: Quarterly report\n")
	assert.Contains(t, out, "Type: pdf\n")
	assert.Contains(t, out, "Size: 59.2 KB\n")
	assert.Contains(t, out, "Download: slck files download F0ABC123\n")
	// No multi-space alignment leaks in.
	assert.NotContains(t, out, "ID:        ")
}

func TestRunGet_AcceptsURL(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "F0ABC123", r.URL.Query().Get("file"))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":   true,
			"file": map[string]interface{}{"id": "F0ABC123", "name": "x", "size": 0},
		})
	})
	defer server.Close()

	require.NoError(t, runGet("https://files.slack.com/files-pri/T1-F0ABC123/x.pdf", c))
}

func TestRunGet_OmitsEmptyTitleAndType(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":   true,
			"file": map[string]interface{}{"id": "F1", "name": "x.bin", "size": 100},
		})
	})
	defer server.Close()

	out := captureOutput(t, func() { require.NoError(t, runGet("F1", c)) })
	assert.NotContains(t, out, "Title:")
	assert.NotContains(t, out, "Type:")
}

func TestRunGet_InvalidRef(t *testing.T) {
	err := runGet("not-a-ref", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not resolve file ID")
}

func TestRunGet_JSONOutputRaw(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id": "F0ABC123", "name": "report.pdf", "title": "Q1", "filetype": "pdf", "size": 100,
			},
		})
	})
	defer server.Close()

	origFmt := output.OutputFormat
	output.OutputFormat = output.FormatJSON
	defer func() { output.OutputFormat = origFmt }()

	out := captureOutput(t, func() { require.NoError(t, runGet("F0ABC123", c)) })

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	// Raw upstream-shaped File. No synthetic ref/hint/download keys.
	assert.Equal(t, "F0ABC123", parsed["id"])
	assert.NotContains(t, parsed, "ref")
	assert.NotContains(t, parsed, "download")
	assert.NotContains(t, parsed, "hint")
}

func TestRunGet_NegativeSizeRendersZero(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":   true,
			"file": map[string]interface{}{"id": "F1", "name": "snippet", "size": -1},
		})
	})
	defer server.Close()

	out := captureOutput(t, func() { require.NoError(t, runGet("F1", c)) })
	assert.Contains(t, out, "Size: 0 B\n")
}

func TestRunGet_APIErrorPropagates(t *testing.T) {
	c, server := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "file_not_found"})
	})
	defer server.Close()

	err := runGet("F0ABC123", c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file_not_found")
}
