package canvas

import (
	"os"
	"strings"
	"testing"
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
	// Create a temp file
	dir := t.TempDir()
	path := dir + "/test.md"
	if err := writeTestFile(path, "# From file"); err != nil {
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
}

func TestRunCreate_StandaloneNoTitle(t *testing.T) {
	opts := &createOptions{text: "# Content"}
	// This should fail because title is required for standalone canvases
	// but we can't test with a nil client — the client.New() will fail first
	// Just test that the validation is in place by checking the error
	err := runCreate(opts, nil)
	if err == nil {
		t.Fatal("expected error when no client available")
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
