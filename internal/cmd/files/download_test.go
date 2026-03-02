package files

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func TestResolveFileID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"bare file ID", "F0AHF3NUSQK", "F0AHF3NUSQK"},
		{"short file ID", "F123", "F123"},
		{"url_private", "https://files.slack.com/files-pri/TNVBX1L3S-F0AHF3NUSQK/image.png", "F0AHF3NUSQK"},
		{"url_private_download", "https://files.slack.com/files-pri/TNVBX1L3S-F0AHF3NUSQK/download/image.png", "F0AHF3NUSQK"},
		{"permalink", "https://signalft.slack.com/files/U099RPJJFRS/F0AHF3NUSQK/image.png", "F0AHF3NUSQK"},
		{"invalid input", "not-a-file-id", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveFileID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRunDownload_Success(t *testing.T) {
	fileContent := "hello world file content"

	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
		w.Write([]byte(fileContent))
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files.info":
			assert.Equal(t, "F123ABC", r.URL.Query().Get("file"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"file": map[string]interface{}{
					"id":                   "F123ABC",
					"name":                 "test-file.txt",
					"size":                 len(fileContent),
					"url_private_download": downloadServer.URL + "/download/test-file.txt",
				},
			})
		default:
			t.Errorf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	destPath := filepath.Join(t.TempDir(), "downloaded.txt")
	opts := &downloadOptions{outputPath: destPath}

	err := runDownload("F123ABC", opts, c)
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, string(data))
}

func TestRunDownload_FromURL(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("image data"))
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files.info":
			assert.Equal(t, "F0AHF3NUSQK", r.URL.Query().Get("file"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok": true,
				"file": map[string]interface{}{
					"id":                   "F0AHF3NUSQK",
					"name":                 "screenshot.png",
					"size":                 10,
					"url_private_download": downloadServer.URL + "/download/screenshot.png",
				},
			})
		}
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	destPath := filepath.Join(t.TempDir(), "screenshot.png")
	opts := &downloadOptions{outputPath: destPath}

	err := runDownload("https://files.slack.com/files-pri/TNVBX1L3S-F0AHF3NUSQK/image.png", opts, c)
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, "image data", string(data))
}

func TestRunDownload_DefaultFilename(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F123",
				"name":                 "report.pdf",
				"size":                 7,
				"url_private_download": downloadServer.URL + "/download",
			},
		})
	}))
	defer server.Close()

	// Change to temp dir so default filename goes there
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &downloadOptions{} // no output path — should use filename from API

	err := runDownload("F123", opts, c)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "report.pdf"))
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestRunDownload_InvalidFileID(t *testing.T) {
	opts := &downloadOptions{}
	err := runDownload("not-a-file", opts, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not resolve file ID")
}

func TestRunDownload_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":                   "F123",
				"name":                 "doc.txt",
				"size":                 42,
				"url_private_download": "https://files.slack.com/download",
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &downloadOptions{outputPath: "/tmp/doc.txt"}

	output.OutputFormat = output.FormatJSON
	defer func() { output.OutputFormat = output.FormatText }()

	var buf strings.Builder
	output.Writer = &buf
	defer func() { output.Writer = os.Stdout }()

	err := runDownload("F123", opts, c)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(buf.String()), &result)
	require.NoError(t, err)

	assert.Equal(t, "F123", result["file_id"])
	assert.Equal(t, "doc.txt", result["name"])
	assert.Equal(t, "/tmp/doc.txt", result["path"])
}

func TestRunDownload_FallsBackToURLPrivate(t *testing.T) {
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("content"))
	}))
	defer downloadServer.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"file": map[string]interface{}{
				"id":          "F123",
				"name":        "file.txt",
				"size":        7,
				"url_private": downloadServer.URL + "/file.txt",
				// no url_private_download
			},
		})
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	destPath := filepath.Join(t.TempDir(), "file.txt")
	opts := &downloadOptions{outputPath: destPath}

	err := runDownload("F123", opts, c)
	require.NoError(t, err)

	data, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}
