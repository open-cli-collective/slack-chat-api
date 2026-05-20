// Package config holds slck's non-secret on-disk configuration per the Open
// CLI Collective Secret-Handling Standard §1.2 / §2.4 and the
// working-with-state.md state-component standard. No access secret is ever
// written here — secrets live only in the OS keyring via cli-common's
// credstore. This file owns config.yml: the authoritative credential_ref
// (§1.3), the non-secret workspace identifier, and the optional §1.4
// file-backend opt-in.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-cli-collective/cli-common/statedir"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultCredentialRef applies when config.yml is absent or omits
	// credential_ref. It is still parsed through credstore.ParseRef by
	// callers — never assumed structurally (§1.3).
	DefaultCredentialRef = "slack-chat-api/default"

	// AppDirName is the credential-scope name for this CLI (working-with-
	// state.md §3). Single-binary → tool/repo name. Exposed so the
	// relocation helpers can compose paths off it.
	AppDirName     = "slack-chat-api"
	configFileName = "config.yml"

	// DirPerm / FilePerm match working-with-state.md §3 (0700 dirs, 0600
	// files). slck was already at these perms; statedir's ConfigDirEnsured
	// enforces 0700 on new creations.
	DirPerm  = 0o700
	FilePerm = 0o600
)

// configScope is the cli-common state-scope for slck's config dir. Resolution
// is os.UserConfigDir + AppDirName: Linux $XDG_CONFIG_HOME (or ~/.config),
// macOS ~/Library/Application Support, Windows %APPDATA%. A relative
// $XDG_CONFIG_HOME on Linux now yields an error (§1.1 intentional tightening).
var configScope = statedir.Scope{Name: AppDirName}

// Config is slck's config.yml. Everything here is safe for an org to commit
// to a private/MDM-controlled store (§1.2); none of it is an access secret.
type Config struct {
	// CredentialRef is the authoritative <service>/<profile> keyring ref
	// (§1.3). Callers resolve it via credstore.ParseRef; the service and
	// profile are never hard-coded from a convention.
	CredentialRef string `yaml:"credential_ref"`
	// Workspace is the non-secret Slack workspace identifier captured at
	// init (verification by `slck me`, shown in `config show`, assertable
	// by org deployment scripts). Not needed for API calls.
	Workspace string `yaml:"workspace,omitempty"`
	// Keyring carries the optional §1.4 explicit file-backend opt-in.
	Keyring KeyringConfig `yaml:"keyring,omitempty"`
}

// KeyringConfig is the §1.4 backend selector. Backend == "file" forces the
// encrypted-file backend unconditionally (the supported path for users who
// genuinely prefer it / headless Linux); empty means OS default selection.
type KeyringConfig struct {
	Backend string `yaml:"backend,omitempty"`
}

// Dir resolves the config directory WITHOUT creating it. Delegated to
// cli-common/statedir; native per OS. A relative $XDG_CONFIG_HOME on Linux
// is rejected (§1.1).
func Dir() (string, error) {
	return configScope.ConfigDir()
}

// dirEnsured resolves and creates the config dir at 0700. Used by Save.
func dirEnsured() (string, error) {
	return configScope.ConfigDirEnsured()
}

// Path is the config.yml location (resolved but not created).
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads config.yml. The strict variant — used by `slck init`'s
// relocation gate. Returns ErrRelocationConflict (with a wrapped detail
// message) when both the old hand-rolled and new statedir-resolved dirs
// contain materially-different config.yml files.
//
// DUAL-RETURN CONTRACT: on a relocation conflict, Load can return BOTH a
// non-nil *Config (the canonical new-dir config, if readable) AND a non-nil
// error wrapping ErrRelocationConflict — so the caller can choose to
// soft-degrade. The standard `if err != nil { return err }` idiom would
// silently DISCARD a valid config in the conflict case; callers must check
// `errors.Is(err, ErrRelocationConflict)` explicitly, or use the
// LoadForRuntime wrapper which encodes the right discipline. Runtime call
// sites have all been migrated to LoadForRuntime; init uses Load directly
// only AFTER its own DetectConfigRelocation gate succeeded (so no conflict
// can be observed at that callsite). If Load couldn't populate cfg (e.g.
// malformed canonical YAML), returns (nil, ErrRelocationConflict) so
// LoadForRuntime hard-fails instead of warning-and-defaulting (MON-5371
// lesson). An absent file is not an error: defaults are applied and a
// usable Config is returned.
func Load() (*Config, error) {
	newDir, err := Dir()
	if err != nil {
		return nil, err
	}
	return loadFromNewDir(newDir)
}

// loadFromNewDir is the testable seam Load and the tests both call. Taking
// an injected newDir is the only way to exercise the divergent/malformed
// branches on Linux where os.UserConfigDir == $XDG_CONFIG_HOME (and so
// would collapse old == new at the public Load entry point).
func loadFromNewDir(newDir string) (*Config, error) {
	relErr := error(nil)
	reloc, derr := detectRelocation(newDir)
	if derr != nil && errors.Is(derr, ErrRelocationConflict) {
		relErr = derr
	} else if derr != nil {
		return nil, derr
	}

	c := &Config{}
	read := false
	// Attempt new-dir read unconditionally so soft-degrading callers get the
	// user's actual settings alongside relErr. Under conflict we don't
	// re-surface a bare parse error (LoadForRuntime needs the wrapped
	// ErrRelocationConflict so it can soft-degrade), but we DO append the
	// parse diagnostic to relErr so the user sees both the conflict path
	// and the corruption detail in one message.
	if reloc.NewPath != "" {
		newYML := filepath.Join(reloc.NewPath, configFileName)
		if data, err := os.ReadFile(newYML); err == nil { //nolint:gosec // path from validated dir
			if uerr := yaml.Unmarshal(data, c); uerr != nil {
				if relErr == nil {
					return nil, fmt.Errorf("parse config %s: %w", newYML, uerr)
				}
				relErr = fmt.Errorf("%w; canonical also malformed: %v", relErr, uerr)
			} else {
				read = true
			}
		} else if !os.IsNotExist(err) {
			if relErr == nil {
				return nil, fmt.Errorf("read config %s: %w", newYML, err)
			}
			relErr = fmt.Errorf("%w; canonical also unreadable: %v", relErr, err)
		}
	}
	if !read && reloc.Kind == relocOldOnly && reloc.OldPath != "" {
		oldYML := filepath.Join(reloc.OldPath, configFileName)
		if data, err := os.ReadFile(oldYML); err == nil { //nolint:gosec // hand-rolled legacy dir
			if uerr := yaml.Unmarshal(data, c); uerr != nil {
				if relErr == nil {
					return nil, fmt.Errorf("parse config %s: %w", oldYML, uerr)
				}
				relErr = fmt.Errorf("%w; old also malformed: %v", relErr, uerr)
			} else {
				read = true
			}
		} else if !os.IsNotExist(err) {
			if relErr == nil {
				return nil, fmt.Errorf("read config %s: %w", oldYML, err)
			}
			relErr = fmt.Errorf("%w; old also unreadable: %v", relErr, err)
		}
	}
	// Malformed canonical under conflict: soft-degrade is unsafe (would swap
	// CredentialRef back to default and mask the corruption). Return nil.
	if !read && relErr != nil {
		return nil, relErr
	}
	c.applyDefaults()
	return c, relErr
}

func (c *Config) applyDefaults() {
	if c.CredentialRef == "" {
		c.CredentialRef = DefaultCredentialRef
	}
}

// Save writes config.yml atomically (temp+chmod+rename) at 0600 under a 0700
// directory. Non-secret, but there is no reason for it to be world-readable.
// Unique temp name from os.CreateTemp so same-process concurrent saves and
// crash-leftover orphans never collide; orphan-tmp from a host crash is
// harmless (never read as config).
func (c *Config) Save() error {
	dir, err := dirEnsured()
	if err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	final := filepath.Join(dir, configFileName)
	tmp, err := os.CreateTemp(dir, "config-*.yml.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp config file: %w", err)
	}
	if err := os.Chmod(tmpPath, FilePerm); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("set config file mode: %w", err)
	}
	if err := os.Rename(tmpPath, final); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("finalize config %s: %w", final, err)
	}
	return nil
}
