package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

func hermeticConfig(t *testing.T) string {
	t.Helper()
	return statedirtest.Hermetic(t)
}

func TestDir_ResolvedUnderUserConfigDir(t *testing.T) {
	hermeticConfig(t)
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	want := filepath.Join(base, AppDirName)
	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if got != want {
		t.Errorf("Dir = %q, want %q", got, want)
	}
}

func TestPath_PointsAtConfigYAML(t *testing.T) {
	hermeticConfig(t)
	dir, _ := Dir()
	p, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if want := filepath.Join(dir, configFileName); p != want {
		t.Errorf("Path = %q, want %q", p, want)
	}
}

func TestLoad_AbsentReturnsDefaults(t *testing.T) {
	hermeticConfig(t)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.CredentialRef != DefaultCredentialRef {
		t.Errorf("CredentialRef = %q, want %q", c.CredentialRef, DefaultCredentialRef)
	}
}

func TestSave_AtomicNoStaleTmp(t *testing.T) {
	hermeticConfig(t)
	cfg := &Config{CredentialRef: "slack-chat-api/test", Workspace: "T_X"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	final, _ := Path()
	info, err := os.Stat(final)
	if err != nil {
		t.Fatalf("stat final: %v", err)
	}
	if info.Mode().Perm() != FilePerm {
		t.Errorf("file mode = %v, want %v", info.Mode().Perm(), FilePerm)
	}
	dir, _ := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("stale temp file left in %s: %s", dir, e.Name())
		}
	}
}

func TestSave_RoundTrip(t *testing.T) {
	hermeticConfig(t)
	in := &Config{CredentialRef: "slack-chat-api/rt", Workspace: "T_Y", Keyring: KeyringConfig{Backend: "file"}}
	if err := in.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.CredentialRef != in.CredentialRef || out.Workspace != in.Workspace || out.Keyring.Backend != in.Keyring.Backend {
		t.Errorf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}
