package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

// reloctest gives two distinct directories (old, new) under a per-test
// hermetic root. detectAt below pins XDG_CONFIG_HOME at the old dir's
// parent so oldHandRolledConfigDir resolves to oldDir on Linux too.
func reloctest(t *testing.T) (oldDir, newDir string) {
	t.Helper()
	root := statedirtest.Hermetic(t)
	oldDir = filepath.Join(root, "old", AppDirName)
	newDir = filepath.Join(root, "new", AppDirName)
	return oldDir, newDir
}

func detectAt(t *testing.T, oldDir, newDir string) (SharedRelocation, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(oldDir))
	return detectRelocation(newDir)
}

func TestRelocate_OldOnly_CopiedAtInit(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, configFileName),
		[]byte("credential_ref: slack-chat-api/old\nworkspace: T_OLD\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocOldOnly || !r.CopyNeeded {
		t.Fatalf("kind=%v copy=%v, want relocOldOnly w/ copy", r.Kind, r.CopyNeeded)
	}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, configFileName)); err != nil {
		t.Errorf("config.yml not copied to new: %v", err)
	}
	// Old preserved (recovery point).
	if _, err := os.Stat(filepath.Join(oldDir, configFileName)); err != nil {
		t.Errorf("old config.yml must remain (copy-leave-old): %v", err)
	}
}

func TestRelocate_NewOnly_LeftUntouched(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, configFileName),
		[]byte("credential_ref: slack-chat-api/only-new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone || r.CopyNeeded {
		t.Fatalf("kind=%v copy=%v, want relocNone", r.Kind, r.CopyNeeded)
	}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(newDir, configFileName))
	if !strings.Contains(string(data), "slack-chat-api/only-new") {
		t.Errorf("new must be untouched, got %q", string(data))
	}
}

func TestRelocate_Equal_NoOp(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, configFileName),
			[]byte("credential_ref: slack-chat-api/same\nworkspace: T1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocBothEqual {
		t.Errorf("kind=%v, want relocBothEqual", r.Kind)
	}
	if r.CopyNeeded {
		t.Errorf("equal must not request a copy")
	}
}

func TestRelocate_Equal_DefaultOmittedVsExplicit_IsEqual(t *testing.T) {
	// Old omits credential_ref (semantically DefaultCredentialRef via
	// applyDefaults); new spells it out. Must classify as equal — not a
	// real user-divergence.
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldDir, configFileName),
		[]byte("workspace: T1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, configFileName),
		[]byte("credential_ref: "+DefaultCredentialRef+"\nworkspace: T1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocBothEqual {
		t.Errorf("kind=%v, want relocBothEqual (omitted == explicit default)", r.Kind)
	}
}

func TestRelocate_Divergent_FailLoudNamesBothPaths_MutatesNothing(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, p := range []struct{ dir, content string }{
		{oldDir, "credential_ref: slack-chat-api/old\n"},
		{newDir, "credential_ref: slack-chat-api/new\n"},
	} {
		if err := os.MkdirAll(p.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, configFileName), []byte(p.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	oldBefore, _ := os.ReadFile(filepath.Join(oldDir, configFileName))
	newBefore, _ := os.ReadFile(filepath.Join(newDir, configFileName))

	r, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("want ErrRelocationConflict, got %v", err)
	}
	if r.Kind != relocBothDivergent {
		t.Errorf("kind=%v, want relocBothDivergent", r.Kind)
	}
	if !strings.Contains(err.Error(), oldDir) || !strings.Contains(err.Error(), newDir) {
		t.Errorf("error must name both paths: %v", err)
	}
	oldAfter, _ := os.ReadFile(filepath.Join(oldDir, configFileName))
	newAfter, _ := os.ReadFile(filepath.Join(newDir, configFileName))
	if string(oldBefore) != string(oldAfter) || string(newBefore) != string(newAfter) {
		t.Errorf("detect must mutate nothing")
	}
}

func TestRelocate_OldOnlyMalformed_FailLoud_MutatesNothing(t *testing.T) {
	oldDir, newDir := reloctest(t)
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, configFileName),
		[]byte("not-valid-yaml: : :\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("malformed old-only must fail loud, got %v", err)
	}
	if r.CopyNeeded {
		t.Errorf("CopyNeeded must be false on malformed old-only")
	}
	if _, e := os.Stat(newDir); !os.IsNotExist(e) {
		t.Errorf("new dir must not exist after malformed-old detect: stat err=%v", e)
	}
}

func TestRelocate_MalformedNew_FailLoud(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(oldDir, configFileName),
		[]byte("credential_ref: slack-chat-api/old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, configFileName),
		[]byte("not-valid-yaml: : :\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := detectAt(t, oldDir, newDir)
	if !errors.Is(err, ErrRelocationConflict) {
		t.Fatalf("malformed new must fail loud, got %v", err)
	}
}

func TestRelocate_Neither_PathResolvedNotCreated(t *testing.T) {
	oldDir, newDir := reloctest(t)
	r, err := detectAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone {
		t.Errorf("kind=%v, want relocNone", r.Kind)
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("detect must not create old dir; stat err=%v", err)
	}
	if _, err := os.Stat(newDir); !os.IsNotExist(err) {
		t.Errorf("detect must not create new dir; stat err=%v", err)
	}
}

func TestRelocate_LinuxOldEqualsNew_ShortCircuits(t *testing.T) {
	root := statedirtest.Hermetic(t)
	same := filepath.Join(root, "same")
	if err := os.MkdirAll(same, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", same)
	r, err := detectRelocation(filepath.Join(same, AppDirName))
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if r.Kind != relocNone {
		t.Errorf("kind=%v, want relocNone on path-identity short-circuit", r.Kind)
	}
}

func TestApplyConfigRelocation_IdempotentSkipsExistingNew(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, d := range []string{oldDir, newDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(newDir, configFileName),
		[]byte("credential_ref: slack-chat-api/new\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, configFileName),
		[]byte("credential_ref: slack-chat-api/old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := SharedRelocation{Kind: relocOldOnly, OldPath: oldDir, NewPath: newDir, CopyNeeded: true}
	if err := ApplyConfigRelocation(r); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got, _ := os.ReadFile(filepath.Join(newDir, configFileName))
	if !strings.Contains(string(got), "slack-chat-api/new") {
		t.Errorf("existing new must not be overwritten, got %q", string(got))
	}
}

// loadForRuntimeAt mirrors Load + LoadForRuntime against injected dirs so
// the soft-degrade vs hard-fail branches are exercisable on Linux.
func loadForRuntimeAt(t *testing.T, oldDir, newDir string) (*Config, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(oldDir))
	reloConflictOnce = sync.Once{}

	r, derr := detectRelocation(newDir)
	relErr := error(nil)
	if derr != nil && errors.Is(derr, ErrRelocationConflict) {
		relErr = derr
	} else if derr != nil {
		return nil, derr
	}
	cfg := &Config{}
	read := false
	if r.NewPath != "" {
		newYML := filepath.Join(r.NewPath, configFileName)
		if data, err := os.ReadFile(newYML); err == nil {
			c, lerr := loadConfigFromFile(newYML)
			if lerr != nil {
				if relErr == nil {
					return nil, lerr
				}
			} else {
				cfg = &c
				read = true
			}
			_ = data
		}
	}
	if !read && r.Kind == relocOldOnly && r.OldPath != "" {
		oldYML := filepath.Join(r.OldPath, configFileName)
		c, lerr := loadConfigFromFile(oldYML)
		if lerr != nil {
			if relErr == nil {
				return nil, lerr
			}
		} else {
			cfg = &c
			read = true
		}
	}
	if !read && relErr != nil {
		return nil, relErr
	}
	cfg.applyDefaults()
	if relErr != nil {
		warnReloConflictOnce(relErr)
		return cfg, nil
	}
	return cfg, nil
}

func TestLoadForRuntime_SoftConflict_ReturnsCanonical(t *testing.T) {
	oldDir, newDir := reloctest(t)
	for _, p := range []struct{ dir, content string }{
		{oldDir, "credential_ref: slack-chat-api/old\n"},
		{newDir, "credential_ref: slack-chat-api/new\nworkspace: T_NEW\n"},
	} {
		if err := os.MkdirAll(p.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, configFileName), []byte(p.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadForRuntimeAt(t, oldDir, newDir)
	if err != nil {
		t.Fatalf("soft-conflict must return nil err, got %v", err)
	}
	if cfg.CredentialRef != "slack-chat-api/new" {
		t.Errorf("soft-conflict must return new-dir cfg, got CredentialRef=%q", cfg.CredentialRef)
	}
	if cfg.Workspace != "T_NEW" {
		t.Errorf("soft-conflict must return non-default fields too, got Workspace=%q", cfg.Workspace)
	}
}

func TestLoadForRuntime_MalformedCanonicalUnderConflict_HardFails(t *testing.T) {
	// Both old and new present; new is malformed. Runtime must hard-fail
	// (not warn-and-default) so a corrupt canonical doesn't silently swap
	// CredentialRef back to the default and mask the corruption.
	oldDir, newDir := reloctest(t)
	for _, p := range []struct{ dir, content string }{
		{oldDir, "credential_ref: slack-chat-api/old\n"},
		{newDir, "not-valid-yaml: : :\n"},
	} {
		if err := os.MkdirAll(p.dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p.dir, configFileName), []byte(p.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadForRuntimeAt(t, oldDir, newDir)
	if err == nil {
		t.Fatalf("malformed canonical under conflict must hard-fail, got cfg=%+v err=nil", cfg)
	}
	if !errors.Is(err, ErrRelocationConflict) {
		t.Errorf("error must wrap ErrRelocationConflict, got %v", err)
	}
	if cfg != nil {
		t.Errorf("cfg must be nil on hard-fail, got %+v", cfg)
	}
}
