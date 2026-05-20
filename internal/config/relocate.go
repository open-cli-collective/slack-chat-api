package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"gopkg.in/yaml.v3"
)

// ErrRelocationConflict is returned by Load (and surfaced through
// LoadForRuntime) when both the old hand-rolled config dir and the new
// statedir-resolved config dir contain a config.yml with materially-different
// user settings. Mutation-free: nothing is copied, nothing is overwritten.
// The user reconciles by running `slck init` (which fails the same way at its
// pre-write gate) or by manually deleting one side.
var ErrRelocationConflict = errors.New("config: shared old/new config diverge")

// relocKind is the four-way classification used by the relocation detector.
// Linux always collapses to relocNone because old and new paths are identical
// (os.UserConfigDir on Linux ≡ $XDG_CONFIG_HOME else $HOME/.config).
type relocKind int

const (
	relocNone          relocKind = iota // old absent OR old==new (Linux short-circuit)
	relocOldOnly                        // only the old hand-rolled config.yml exists
	relocBothEqual                      // both exist with materially-equal Configs
	relocBothDivergent                  // both exist with materially-different Configs
)

// SharedRelocation is the result of DetectConfigRelocation. Paths are filled
// even on relocNone so callers can log/diagnose; CopyNeeded is true iff a
// gated ApplyConfigRelocation would actually do work.
type SharedRelocation struct {
	Kind       relocKind
	OldPath    string // old hand-rolled config dir; "" on no-HOME edge
	NewPath    string // statedir-resolved config dir
	CopyNeeded bool   // relocOldOnly only
}

// oldHandRolledConfigDir reproduces the prior pre-MON-5372 resolver:
// $XDG_CONFIG_HOME if set, else $HOME/.config; then "/slack-chat-api". Same
// shape on Linux/macOS/Windows (the deliberate "no %APPDATA% branch"). A
// missing HOME is an error (matches the original behavior).
func oldHandRolledConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, AppDirName), nil
}

// OldConfigPath returns the pre-MON-5372 hand-rolled config.yml location.
// Exported so `slck config clear --all` can scrub it alongside the new path
// (runtime old-only fallback would otherwise let a stale old file
// silently resurrect the config post-clear).
func OldConfigPath() (string, error) {
	dir, err := oldHandRolledConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// DetectConfigRelocation classifies the old/new pair without touching disk
// beyond stats and reads. Never copies, never writes. On Linux (old==new) it
// short-circuits to relocNone.
func DetectConfigRelocation() (SharedRelocation, error) {
	newDir, err := Dir()
	if err != nil {
		return SharedRelocation{}, err
	}
	return detectRelocation(newDir)
}

// detectRelocation is the testable core: the new-dir path is injected so
// macOS/Windows divergence can be exercised on Linux CI.
func detectRelocation(newDir string) (SharedRelocation, error) {
	oldDir, err := oldHandRolledConfigDir()
	if err != nil {
		// Old path unresolvable (no HOME): treat as relocNone with new-only.
		// Load still works against new; the gate is harmless.
		return SharedRelocation{Kind: relocNone, NewPath: newDir}, nil
	}
	if oldDir == newDir {
		// Linux: identical paths — nothing to relocate.
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	}

	oldYML := filepath.Join(oldDir, configFileName)
	newYML := filepath.Join(newDir, configFileName)
	oldPresent := fileExists(oldYML)
	newPresent := fileExists(newYML)
	switch {
	case !oldPresent && !newPresent:
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	case oldPresent && !newPresent:
		// Validate old parses cleanly BEFORE signaling CopyNeeded — otherwise
		// init's ApplyConfigRelocation would propagate a malformed legacy
		// file into the new dir and Load would die parsing it post-copy.
		// §3.2 malformed-old: fail loud, mutate nothing. (MON-5371 lesson.)
		if _, oerr := loadConfigFromFile(oldYML); oerr != nil {
			return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
				fmt.Errorf("%w: old %s is malformed: %w", ErrRelocationConflict, oldYML, oerr)
		}
		return SharedRelocation{Kind: relocOldOnly, OldPath: oldDir, NewPath: newDir, CopyNeeded: true}, nil
	case !oldPresent && newPresent:
		return SharedRelocation{Kind: relocNone, OldPath: oldDir, NewPath: newDir}, nil
	}

	// Both present — load and compare comparable subset.
	oldCfg, oerr := loadConfigFromFile(oldYML)
	newCfg, nerr := loadConfigFromFile(newYML)
	if oerr != nil {
		return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
			fmt.Errorf("%w: old %s unreadable: %w", ErrRelocationConflict, oldYML, oerr)
	}
	if nerr != nil {
		return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
			fmt.Errorf("%w: new %s unreadable: %w", ErrRelocationConflict, newYML, nerr)
	}
	if configsMaterialEqual(oldCfg, newCfg) {
		return SharedRelocation{Kind: relocBothEqual, OldPath: oldDir, NewPath: newDir}, nil
	}
	return SharedRelocation{Kind: relocBothDivergent, OldPath: oldDir, NewPath: newDir},
		fmt.Errorf("%w: old %s and new %s have different settings; reconcile (or delete one) before running slck init",
			ErrRelocationConflict, oldYML, newYML)
}

// loadConfigFromFile parses a single config.yml. applyDefaults is NOT applied
// here directly — configsMaterialEqual applies it to both sides before
// comparing, so omitted-vs-explicit DefaultCredentialRef classify as equal.
func loadConfigFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path composed from validated config dir
	if err != nil {
		return Config{}, err
	}
	var c Config
	if uerr := yaml.Unmarshal(data, &c); uerr != nil {
		return Config{}, uerr
	}
	return c, nil
}

// configsMaterialEqual compares two Configs after applying defaults on both
// sides (so an omitted `credential_ref` — semantically DefaultCredentialRef
// — compares equal to an explicit DefaultCredentialRef). reflect.DeepEqual
// on the whole default-applied struct so any future Config field is
// automatically covered as a divergence-trigger; we don't need to remember
// to update this comparator when Config grows.
func configsMaterialEqual(a, b Config) bool {
	a.applyDefaults()
	b.applyDefaults()
	return reflect.DeepEqual(a, b)
}

// ApplyConfigRelocation copies the single config.yml file from old → new
// atomically; idempotent (skips if new already exists). The old dir is NOT
// modified — leave-old gives the user a recovery point and matches the
// MON-5370/5371 family invariant. Called only from `slck init`'s gate.
func ApplyConfigRelocation(r SharedRelocation) error {
	if !r.CopyNeeded {
		return nil
	}
	if r.OldPath == "" || r.NewPath == "" {
		return fmt.Errorf("config: ApplyConfigRelocation called with empty path")
	}
	if err := os.MkdirAll(r.NewPath, DirPerm); err != nil {
		return fmt.Errorf("create new config dir: %w", err)
	}
	src := filepath.Join(r.OldPath, configFileName)
	dst := filepath.Join(r.NewPath, configFileName)
	if fileExists(dst) {
		return nil // idempotent
	}
	return copyFileAtomic(src, dst)
}

func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src) //nolint:gosec // path from old config dir
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, FilePerm); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// fileExists distinguishes "not present" from other stat errors. A
// permission-denied (or any other non-IsNotExist) error must NOT silently
// degrade to "absent" — that would let an oddly-permissioned old dir
// collapse an old-only relocation to a no-op. Treat unknown errors as
// "present" so the relocation flow's subsequent open/read surfaces the
// real error instead of skipping the gate.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return !os.IsNotExist(err)
}

// LoadForRuntime is the soft-conflict variant of Load for non-init callers.
// On ErrRelocationConflict it prints a one-shot stderr warning and returns
// the canonical (new-dir) config so the command can keep working — BUT only
// when a canonical config was actually read. If Load couldn't populate cfg
// (e.g. malformed YAML on the canonical side), the runtime must hard-fail
// instead of warning-and-defaulting; otherwise it would silently swap
// CredentialRef back to DefaultCredentialRef and mask the corrupt file
// (MON-5371 contract).
func LoadForRuntime() (*Config, error) {
	newDir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loadForRuntimeFromNewDir(newDir)
}

// loadForRuntimeFromNewDir is the testable seam LoadForRuntime and the
// tests both call.
func loadForRuntimeFromNewDir(newDir string) (*Config, error) {
	cfg, err := loadFromNewDir(newDir)
	if err != nil && errors.Is(err, ErrRelocationConflict) && cfg != nil {
		warnReloConflictOnce(err)
		return cfg, nil
	}
	return cfg, err
}

var reloConflictOnce sync.Once

func warnReloConflictOnce(err error) {
	reloConflictOnce.Do(func() {
		fmt.Fprintf(os.Stderr, "warning: %v; using the new config. Run `slck init` to reconcile.\n", err)
	})
}
