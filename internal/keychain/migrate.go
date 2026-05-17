package keychain

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// One-time legacy migration (§1.8 / §2.4). Reads any legacy credential
// location that still exists, writes the new credstore bundle, surfaces the
// signal (stderr for humans, _migration for JSON), then deletes the legacy
// originals. Idempotent: once the originals are gone there is nothing to do
// and no signal fires. Conflicts (legacy vs legacy, or legacy vs an existing
// keyring value) fail loudly per §1.8 — all conflicts are detected before
// any write or delete, and on conflict every legacy and keyring entry is
// left exactly as it was.

// legacy keychain service names slck has used historically (§2.4): the
// current "slack-chat-api" and the older "slck". Account names are the
// legacy field names; "api_token" is renamed to bot_token in the new layout.
var legacyKeychainServices = []string{"slck", "slack-chat-api"}

// legacyFields maps a legacy field/account name to the new bundle key.
var legacyFields = map[string]string{"api_token": KeyBotToken, "user_token": KeyUserToken}

// candidate is one discovered legacy value for a logical new key.
type candidate struct {
	newKey      string // bot_token | user_token
	legacyField string // api_token | user_token (what §1.8 `field` must name)
	location    string // non-secret descriptor (never the value)
	value       string
	deleter     func() error // removes this specific legacy original
}

// migrateLegacyOverwrite runs the one-time migration. overwrite (the §1.8
// `--overwrite` path, reached via keychain.OpenForMigrationOverwrite) forces
// a legacy value over an existing keyring entry; it cannot resolve a
// legacy-vs-legacy disagreement — the user must still pick.
func migrateLegacyOverwrite(s *Store, cfg *config.Config, overwrite bool) error {
	cands := discover(s.service)
	if len(cands) == 0 {
		return nil // nothing legacy on disk/keychain — the steady state
	}

	// Group candidates by new key, in deterministic key order.
	byKey := map[string][]candidate{}
	for _, c := range cands {
		byKey[c.newKey] = append(byKey[c.newKey], c)
	}
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Phase 1: resolve every key, detecting ALL conflicts before mutating.
	writes := map[string]string{}            // newKey -> value to SetBundle
	changes := []credstore.MigrationChange{} // for the _migration block
	humanField := map[string]string{}        // newKey -> legacyField (stderr)
	var cleanups []func() error              // run only after a clean write

	for _, k := range keys {
		group := byKey[k]
		distinct := map[string]bool{}
		for _, c := range group {
			distinct[c.value] = true
		}

		target, hasTarget := currentValue(s, k)

		switch {
		case len(distinct) > 1:
			// Legacy sources disagree among themselves: --overwrite cannot
			// pick a winner here either (§1.8 — the user must). Always a
			// conflict, regardless of overwrite or target presence.
			return conflict(s, group, target, hasTarget)
		case hasTarget && !overwrite && disagrees(distinct, target):
			return conflict(s, group, target, hasTarget)
		case hasTarget && !disagrees(distinct, target):
			// Already migrated (values match): no write, just clean up the
			// leftover originals from an interrupted prior run. No signal.
			for _, c := range group {
				cleanups = append(cleanups, c.deleter)
			}
		default:
			// Resolvable migration: one value (or overwrite forcing one).
			val := group[0].value
			writes[k] = val
			lf := group[0].legacyField
			humanField[k] = lf
			changes = append(changes, credstore.MigrationJSONEntry(
				lf, group[0].location,
				fmt.Sprintf("keyring:%s/%s/%s", s.service, s.profile, k)))
			for _, c := range group {
				cleanups = append(cleanups, c.deleter)
			}
		}
	}

	// Phase 2: write the new bundle (no overwrite needed when target absent;
	// WithOverwrite only when the caller forced it).
	if len(writes) > 0 {
		var opts []credstore.SetOpt
		if overwrite {
			opts = append(opts, credstore.WithOverwrite())
		}
		if _, err := s.cs.SetBundle(s.profile, writes, opts...); err != nil {
			return fmt.Errorf("migrate to keyring %s: %w", s.ref, err)
		}
	}

	// Phase 3: surface the signal (only for keys actually moved this run).
	// Record the _migration block ONLY on a JSON run — recording it on a
	// text run would leave a stale block that a later JSON command in the
	// same (or test) process could splice into an unrelated response.
	if len(changes) > 0 {
		if output.IsJSON() {
			output.RecordMigration(credstore.NewMigrationBlock(changes...))
		} else {
			for _, k := range keys {
				if lf, ok := humanField[k]; ok {
					credstore.EmitMigrationStderr(lf, s.ref)
				}
			}
		}
	}

	// Phase 4: delete the legacy originals, then persist credential_ref so
	// the migration is not re-attempted (§1.8 "adding credential_ref").
	for _, del := range cleanups {
		if err := del(); err != nil {
			return fmt.Errorf("migration wrote the keyring but could not remove a legacy original (%s): %w", s.ref, err)
		}
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("migration succeeded but writing config.yml failed: %w", err)
	}
	return nil
}

// currentValue reports the existing credstore value for a key (post-Open).
func currentValue(s *Store, key string) (string, bool) {
	v, err := s.cs.Get(s.profile, key)
	if err != nil || v == "" {
		return "", false
	}
	return v, true
}

func disagrees(distinct map[string]bool, target string) bool {
	for v := range distinct {
		if v != target {
			return true
		}
	}
	return false
}

// conflict builds the §1.8 error: names every legacy source and the keyring
// target, never a value (masked or not). Reports the legacy field name.
func conflict(s *Store, group []candidate, target string, hasTarget bool) error {
	locs := make([]string, 0, len(group)+1)
	for _, c := range group {
		locs = append(locs, c.location)
	}
	if hasTarget {
		_ = target // value intentionally unused — never printed
		locs = append(locs, fmt.Sprintf("keyring:%s/%s/%s", s.service, s.profile, group[0].newKey))
	}
	return credstore.MigrationConflictError("slck", group[0].legacyField, strings.Join(locs, ", "), s.ref)
}

// legacyKeychainScanDisabledEnv is a test-only seam: when set, discover()
// skips the darwin `security` shell-out so the test suite is hermetic and
// never touches the real login Keychain. Production never sets it — legacy
// Keychain discovery must run regardless of the destination backend
// (§2.4: a macOS user who opts into keyring.backend:file must still have
// old `slck`/`slack-chat-api` Keychain items migrated).
const legacyKeychainScanDisabledEnv = "SLCK_TEST_DISABLE_LEGACY_KEYCHAIN_SCAN"

// discover enumerates every legacy source that currently exists,
// independent of the destination backend (§2.4). macOS Keychain reads are
// migration-only and darwin-only (the sole sanctioned `security`
// shell-out). The plaintext file path is the released layout —
// $XDG_CONFIG_HOME/slack-chat-api/credentials else ~/.config/... — on Linux
// AND Windows alike (the legacy code has no %APPDATA% branch).
func discover(service string) []candidate {
	var out []candidate

	if runtime.GOOS == "darwin" && os.Getenv(legacyKeychainScanDisabledEnv) == "" {
		for _, svc := range legacyKeychainServices {
			for field, newKey := range legacyFields {
				svc, field, newKey := svc, field, newKey
				if v, ok := keychainRead(svc, field); ok {
					out = append(out, candidate{
						newKey:      newKey,
						legacyField: field,
						location:    fmt.Sprintf("keychain:%s/%s", svc, field),
						value:       v,
						deleter:     func() error { return keychainDelete(svc, field) },
					})
				}
			}
		}
	}

	path := legacyCredentialsPath()
	for field, v := range readLegacyFile(path) {
		field, v := field, v
		newKey, known := legacyFields[field]
		if !known {
			continue
		}
		out = append(out, candidate{
			newKey:      newKey,
			legacyField: field,
			location:    fmt.Sprintf("file:%s#%s", path, field),
			value:       v,
			// Secrets-only file: both keys share one deleter target, so it
			// must be idempotent — the first key's cleanup removes the file
			// and the second must treat already-gone as success.
			deleter: func() error {
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
				return nil
			},
		})
	}
	return out
}

func legacyCredentialsPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "slack-chat-api", "credentials")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slack-chat-api", "credentials")
}

// readLegacyFile parses the legacy key=value credentials file. Missing file
// → empty map (not an error: the steady state).
func readLegacyFile(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	m := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if i := strings.IndexByte(line, '='); i > 0 {
			m[line[:i]] = line[i+1:]
		}
	}
	return m
}

func keychainRead(service, account string) (string, bool) {
	out, err := exec.Command("security", "find-generic-password",
		"-s", service, "-a", account, "-w").Output()
	if err != nil {
		return "", false
	}
	v := strings.TrimRight(string(out), "\r\n")
	if v == "" {
		return "", false
	}
	return v, true
}

// securityErrItemNotFound is `security`'s exit status when the item is
// absent (SecKeychainErrorCode errSecItemNotFound). Only this is treated as
// idempotent success — a denial/locked/other failure must surface so we
// don't silently leave a legacy secret behind after "migration".
const securityErrItemNotFound = 44

func keychainDelete(service, account string) error {
	err := exec.Command("security", "delete-generic-password",
		"-s", service, "-a", account).Run()
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == securityErrItemNotFound {
		return nil // already absent — fine for idempotent cleanup
	}
	return fmt.Errorf("remove legacy keychain item %s/%s: %w", service, account, err)
}
