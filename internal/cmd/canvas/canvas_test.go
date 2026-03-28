package canvas

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

func TestResolveContent_Text(t *testing.T) {
	content, err := resolveContent("# Hello", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# Hello" {
		t.Errorf("expected '# Hello', got %q", content)
	}
}

func TestResolveContent_Stdin(t *testing.T) {
	r := strings.NewReader("# From stdin")
	content, err := resolveContent("", "-", r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# From stdin" {
		t.Errorf("expected '# From stdin', got %q", content)
	}
}

func TestResolveContent_BothTextAndFile(t *testing.T) {
	_, err := resolveContent("text", "file.md", nil)
	if err == nil {
		t.Fatal("expected error when both text and file provided")
	}
}

func TestResolveContent_Empty(t *testing.T) {
	content, err := resolveContent("", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty string, got %q", content)
	}
}

func TestResolveContent_File(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/test.md"
	if err := os.WriteFile(path, []byte("# From file"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	content, err := resolveContent("", path, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "# From file" {
		t.Errorf("expected '# From file', got %q", content)
	}
}

func TestResolveContent_FileNotFound(t *testing.T) {
	_, err := resolveContent("", "/nonexistent/file.md", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRunCreate_NoContent(t *testing.T) {
	opts := &createOptions{title: "Test"}
	err := runCreate(opts, nil)
	if err == nil {
		t.Fatal("expected error when no content provided")
	}
	if !strings.Contains(err.Error(), "content required") {
		t.Errorf("expected content required error, got: %v", err)
	}
}

func TestRunCreate_StandaloneNoTitle(t *testing.T) {
	opts := &createOptions{text: "# Content"}
	err := runCreate(opts, nil)
	if err == nil {
		t.Fatal("expected error when no title for standalone canvas")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected title required error, got: %v", err)
	}
}

func TestRunCreate_TitleWithChannel(t *testing.T) {
	opts := &createOptions{text: "# Content", title: "Test", channel: "C123"}
	err := runCreate(opts, nil)
	if err == nil {
		t.Fatal("expected error when title used with channel")
	}
	if !strings.Contains(err.Error(), "--title is not used with --channel") {
		t.Errorf("expected title+channel error, got: %v", err)
	}
}

func TestRunCreate_Standalone_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{"ok": true, "canvas_id": "F12345"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &createOptions{title: "Test Canvas", text: "# Hello"}
	err := runCreate(opts, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCreate_Channel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{"ok": true, "canvas_id": "F67890"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &createOptions{channel: "C12345ABC", text: "# Channel Doc"}
	err := runCreate(opts, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunEdit_NoContent(t *testing.T) {
	opts := &editOptions{}
	err := runEdit("F12345", opts, nil)
	if err == nil {
		t.Fatal("expected error when no content provided")
	}
	if !strings.Contains(err.Error(), "content required") {
		t.Errorf("expected content required error, got: %v", err)
	}
}

func TestRunEdit_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &editOptions{text: "# Updated"}
	err := runEdit("F12345", opts, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunEdit_Stdin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	opts := &editOptions{file: "-", stdin: strings.NewReader("# From stdin")}
	err := runEdit("F12345", opts, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDelete_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := client.NewWithConfig(server.URL, "test-token", nil)
	err := runDelete("F12345", c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
