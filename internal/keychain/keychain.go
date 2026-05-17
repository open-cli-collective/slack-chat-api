// Package keychain is slck's credential adapter. Despite the historical
// package name, it no longer shells out to macOS `security` or writes a
// plaintext file: it is a thin wrapper over cli-common's credstore, which
// owns OS-keyring storage, §1.4 backend selection (incl. Linux fail-closed
// and the encrypted-file fallback), and the §1.5.2 allowed-key allowlist.
// The name is retained only to avoid churning every importer during the
// Phase B pilot (Open CLI Collective Secret-Handling Standard §2.4).
//
// All runtime credential resolution goes through here and reads the OS
// keyring only — never an environment variable, never a config field
// (§1.11 acceptance item 2). Environment variables carry secret material
// into slck solely as *ingress* during `init` / `set-credential`.
package keychain

import (
	"errors"
	"fmt"
	"strings"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/config"
)

// Bundle key names (§2.4). Note bot_token, not the legacy api_token — the
// rename is performed by the one-time migration (migrate.go).
const (
	KeyBotToken  = "bot_token"
	KeyUserToken = "user_token"
)

// allowedKeys is slck's §1.5.2 allowlist: exactly the two bundle keys.
var allowedKeys = []string{KeyBotToken, KeyUserToken}

// Store is an open handle to slck's credential bundle. Construct with Open,
// always Close. It carries the resolved ref so callers can report it in
// `config show` / errors without re-deriving it (§1.12: ref is not secret).
type Store struct {
	cs      *credstore.Store
	service string
	profile string
	ref     string
}

// Open resolves the authoritative credential_ref from config.yml (§1.3 —
// the service/profile are parsed, never assumed), opens the backing
// credstore, and runs the one-time legacy migration (§1.8) before returning.
// The returned Store reads/writes the OS keyring only. A legacy-vs-keyring
// conflict surfaces here as a §1.8 error; `slck init --overwrite` calls
// OpenForMigrationOverwrite to force the legacy value instead.
func Open() (*Store, error) { return open(false) }

// OpenForMigrationOverwrite is Open with the §1.8 `--overwrite` resolution:
// a legacy value is forced over an existing keyring entry. It still cannot
// resolve a legacy-vs-legacy disagreement (the user must pick).
func OpenForMigrationOverwrite() (*Store, error) { return open(true) }

func open(overwrite bool) (*Store, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return openWith(cfg, overwrite)
}

// OpenRef opens a store against an explicit ref instead of config.yml's
// credential_ref — used by `slck set-credential --ref` (§1.5.2 ingress).
// An empty ref falls back to the configured/default ref.
func OpenRef(ref string) (*Store, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if ref != "" {
		cfg.CredentialRef = ref
	}
	return openWith(cfg, false)
}

// openWith is the seam unit tests drive with an injected config (e.g. a
// memory-backend opt-in via Keyring.Backend) so they never touch a real
// keyring (§1.12 test obligation, and hermeticity).
func openWith(cfg *config.Config, overwrite bool) (*Store, error) {
	service, profile, err := credstore.ParseRef(cfg.CredentialRef)
	if err != nil {
		return nil, fmt.Errorf("invalid credential_ref %q: %w", cfg.CredentialRef, err)
	}

	opts := &credstore.Options{AllowedKeys: allowedKeys}
	if b, ok := configBackend(cfg.Keyring.Backend); ok {
		opts.ConfigBackend = b
	}
	opts.FilePassphrase = passphraseFunc(service)

	cs, err := credstore.Open(service, opts)
	if err != nil {
		return nil, err
	}

	s := &Store{cs: cs, service: service, profile: profile, ref: cfg.CredentialRef}

	if err := migrateLegacyOverwrite(s, cfg, overwrite); err != nil {
		_ = cs.Close()
		return nil, err
	}
	return s, nil
}

// configBackend maps the config.yml keyring.backend value to a credstore
// Backend. Only the §1.4 "file" opt-in is honored; anything else is left to
// credstore's fail-closed default selection (ok=false). Empty is the
// common case (auto-select).
func configBackend(v string) (credstore.Backend, bool) {
	if strings.TrimSpace(v) == "file" {
		return credstore.BackendFile, true
	}
	return "", false
}

// Close releases the backing store. Safe on a nil receiver.
func (s *Store) Close() error {
	if s == nil || s.cs == nil {
		return nil
	}
	return s.cs.Close()
}

// Ref returns the resolved credential ref (non-secret; safe to display).
func (s *Store) Ref() string { return s.ref }

// Backend reports the credstore backend and how it was selected, for
// `config show` (§1.11 item 7). Neither value is secret.
func (s *Store) Backend() (credstore.Backend, credstore.Source) { return s.cs.Backend() }

// BotToken returns the bot token from the keyring. ErrMissingBotToken (an
// errors.Is-matchable wrapper of credstore.ErrNotFound) when unset.
func (s *Store) BotToken() (string, error) { return s.get(KeyBotToken, ErrMissingBotToken) }

// UserToken returns the user token from the keyring.
func (s *Store) UserToken() (string, error) { return s.get(KeyUserToken, ErrMissingUserToken) }

func (s *Store) get(key string, missing error) (string, error) {
	v, err := s.cs.Get(s.profile, key)
	if errors.Is(err, credstore.ErrNotFound) || (err == nil && v == "") {
		return "", missing
	}
	if err != nil {
		// Never embed the value; naming ref/key/op is allowed (§1.12).
		return "", fmt.Errorf("read %s from %s: %w", key, s.ref, err)
	}
	return v, nil
}

// SetBotToken / SetUserToken are ingress-only writes (called from init /
// set-credential after the value arrived via stdin/env per §1.5).
func (s *Store) SetBotToken(v string) error  { return s.set(KeyBotToken, v) }
func (s *Store) SetUserToken(v string) error { return s.set(KeyUserToken, v) }

func (s *Store) set(key, v string) error {
	if err := s.cs.Set(s.profile, key, v, credstore.WithOverwrite()); err != nil {
		return fmt.Errorf("store %s at %s: %w", key, s.ref, err)
	}
	return nil
}

// DeleteBotToken / DeleteUserToken remove a single key (idempotent: a
// missing key is not an error — §1.7).
func (s *Store) DeleteBotToken() error  { return s.del(KeyBotToken) }
func (s *Store) DeleteUserToken() error { return s.del(KeyUserToken) }

func (s *Store) del(key string) error {
	// Idempotent by construction (§1.7): an absent key is success. The
	// Exists pre-check is backend-agnostic — credstore's file backend
	// surfaces a raw os "not found" rather than ErrNotFound on Delete.
	if ok, _ := s.cs.Exists(s.profile, key); !ok {
		return nil
	}
	if err := s.cs.Delete(s.profile, key); err != nil && !errors.Is(err, credstore.ErrNotFound) {
		return fmt.Errorf("delete %s at %s: %w", key, s.ref, err)
	}
	return nil
}

// HasBotToken / HasUserToken report presence without returning the value
// (used by `config show` / `init` overwrite prompts — §1.11 item 3).
func (s *Store) HasBotToken() bool  { return s.has(KeyBotToken) }
func (s *Store) HasUserToken() bool { return s.has(KeyUserToken) }

func (s *Store) has(key string) bool {
	ok, err := s.cs.Exists(s.profile, key)
	return err == nil && ok
}

// Clear removes the whole bundle under the active profile (config clear,
// §1.7). Idempotent; scope is the active profile only.
func (s *Store) Clear() ([]string, error) {
	return s.cs.DeleteBundle(s.profile)
}

// Sentinel "missing" errors. errors.Is(err, ErrMissingBotToken) lets the
// CLI print an actionable setup hint without leaking anything.
var (
	ErrMissingBotToken  = errors.New("slck: no bot token in keyring — run `slck init` or `slck set-credential --key bot_token --stdin`")
	ErrMissingUserToken = errors.New("slck: no user token in keyring — run `slck init` or `slck set-credential --key user_token --stdin`")
)

// DetectTokenType returns "bot" for xoxb-*, "user" for xoxp-*, else
// "unknown". Pure; used by init to validate the slot a token was given to.
func DetectTokenType(token string) string {
	switch {
	case strings.HasPrefix(token, "xoxb-"):
		return "bot"
	case strings.HasPrefix(token, "xoxp-"):
		return "user"
	default:
		return "unknown"
	}
}
