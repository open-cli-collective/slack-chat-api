//go:build darwin && cgo

package keychain

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/byteness/keyring"
	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/testutil"
)

func TestKeychainMetadataGated(t *testing.T) {
	if os.Getenv("SLCK_KEYCHAIN_METADATA_TEST") != "1" {
		t.Skip("set SLCK_KEYCHAIN_METADATA_TEST=1 to write to the real macOS Keychain")
	}

	home := os.Getenv("HOME")
	testutil.Setup(t)
	// testutil.Setup isolates HOME for config tests, but the real macOS
	// Keychain backend uses HOME to find the user's default login keychain.
	t.Setenv("HOME", home)
	SetBackendFlagOverride(string(credstore.BackendKeychain), true)
	t.Cleanup(func() { SetBackendFlagOverride("", false) })

	profile := "metadata-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	ref := "slack-chat-api/" + profile
	t.Logf("using synthetic Keychain ref %q", ref)

	st, err := openWith(&config.Config{CredentialRef: ref}, false, false)
	if err != nil {
		t.Fatalf("openWith(%q): %v", ref, err)
	}
	t.Cleanup(func() { _ = st.Close() })
	t.Cleanup(func() { _ = st.DeleteBotToken() })
	t.Cleanup(func() { _ = st.DeleteUserToken() })

	kr, err := keyring.Open(keyring.Config{
		ServiceName:              "slack-chat-api",
		AllowedBackends:          []keyring.BackendType{keyring.KeychainBackend},
		KeychainTrustApplication: true,
	})
	if err != nil {
		t.Fatalf("open ByteNess Keychain backend: %v", err)
	}

	cases := []struct {
		key string
		set func(string) error
	}{
		{key: KeyBotToken, set: st.SetBotToken},
		{key: KeyUserToken, set: st.SetUserToken},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			account := profile + "/" + tc.key
			t.Logf("using synthetic Keychain account %q", account)
			t.Cleanup(func() { _ = kr.Remove(account) })

			wantLabel := "slack-chat-api " + account
			wantDescription := "Credential for slack-chat-api " + account

			if err := tc.set("xox-test-metadata-fresh"); err != nil {
				t.Fatalf("fresh set %s: %v", tc.key, err)
			}
			assertMetadata(t, kr, account, wantLabel, wantDescription)

			if err := kr.Remove(account); err != nil {
				t.Fatalf("remove fresh item before stale seed: %v", err)
			}

			if err := kr.Set(keyring.Item{Key: account, Data: []byte("legacy")}); err != nil {
				t.Fatalf("seed stale metadata item: %v", err)
			}
			seeded, err := kr.GetMetadata(account)
			if err != nil {
				t.Fatalf("GetMetadata(%q) after seed: %v", account, err)
			}
			if seeded.Item == nil {
				t.Fatalf("GetMetadata(%q) after seed returned nil item", account)
			}
			if seeded.Label == wantLabel && seeded.Description == wantDescription {
				t.Fatalf("seeded item already has target metadata label=%q description=%q", seeded.Label, seeded.Description)
			}

			if err := tc.set("xox-test-metadata"); err != nil {
				t.Fatalf("set %s: %v", tc.key, err)
			}

			assertMetadata(t, kr, account, wantLabel, wantDescription)
		})
	}
}

func assertMetadata(t *testing.T, kr keyring.Keyring, account, wantLabel, wantDescription string) {
	t.Helper()

	md, err := kr.GetMetadata(account)
	if err != nil {
		t.Fatalf("GetMetadata(%q): %v", account, err)
	}
	if md.Item == nil {
		t.Fatalf("GetMetadata(%q) returned nil item", account)
	}
	if md.Label != wantLabel {
		t.Fatalf("metadata label = %q, want %q", md.Label, wantLabel)
	}
	if md.Description != wantDescription {
		t.Fatalf("metadata description = %q, want %q", md.Description, wantDescription)
	}
}
