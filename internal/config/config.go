// Package config holds slck's non-secret on-disk configuration per the Open
// CLI Collective Secret-Handling Standard §1.2 / §2.4. No access secret is
// ever written here — secrets live only in the OS keyring via cli-common's
// credstore. This file owns config.yml: the authoritative credential_ref
// (§1.3), the non-secret workspace identifier, and the optional §1.4
// file-backend opt-in.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultCredentialRef applies when config.yml is absent or omits
	// credential_ref. It is still parsed through credstore.ParseRef by
	// callers — never assumed structurally (§1.3).
	DefaultCredentialRef = "slack-chat-api/default"

	appDirName     = "slack-chat-api"
	configFileName = "config.yml"
)

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

// Dir is the cross-platform config directory: $XDG_CONFIG_HOME/slack-chat-api
// else ~/.config/slack-chat-api. Identical on Linux, macOS, and Windows —
// this matches the released layout (the legacy code has no %APPDATA% branch),
// so config.yml sits beside the legacy credentials file it supersedes.
func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appDirName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", appDirName)
}

// Path is the config.yml location.
func Path() string { return filepath.Join(Dir(), configFileName) }

// Load reads config.yml. An absent file is not an error: defaults are applied
// (CredentialRef = DefaultCredentialRef) and a usable Config is returned.
func Load() (*Config, error) {
	c := &Config{}
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			c.applyDefaults()
			return c, nil
		}
		return nil, fmt.Errorf("read config %s: %w", Path(), err)
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", Path(), err)
	}
	c.applyDefaults()
	return c, nil
}

func (c *Config) applyDefaults() {
	if c.CredentialRef == "" {
		c.CredentialRef = DefaultCredentialRef
	}
}

// Save writes config.yml at 0600 under a 0700 directory. Non-secret, but
// there is no reason for it to be world-readable.
func (c *Config) Save() error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(Path(), data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", Path(), err)
	}
	return nil
}
