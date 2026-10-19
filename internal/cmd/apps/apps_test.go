package apps

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/apptoken"
	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

const (
	accessToken  = "xoxe.xoxp-1-STORED-access-canary"
	refreshToken = "xoxe-1-STORED-refresh-canary"
)

var now = time.Unix(1790000000, 0)

// call is one request the fake Slack received.
type call struct {
	method string
	auth   string
	form   map[string]string
}

// fakeSlack answers each method with the fixture named for it; a method
// with several fixtures answers them in turn.
type fakeSlack struct {
	t         *testing.T
	mu        sync.Mutex
	responses map[string][]string
	calls     []call
}

func newFakeSlack(t *testing.T, responses map[string][]string) (*fakeSlack, *client.AppsClient) {
	f := &fakeSlack{t: t, responses: responses}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, client.NewAppsClientWithConfig(srv.URL, nil)
}

func (f *fakeSlack) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	require.NoError(f.t, r.ParseForm())
	method := strings.TrimPrefix(r.URL.Path, "/")
	form := map[string]string{}
	for k := range r.PostForm {
		form[k] = r.PostForm.Get(k)
	}
	f.mu.Lock()
	f.calls = append(f.calls, call{method: method, auth: r.Header.Get("Authorization"), form: form})
	queue := f.responses[method]
	require.NotEmpty(f.t, queue, "unexpected call to %s", method)
	fixture := queue[0]
	if len(queue) > 1 {
		f.responses[method] = queue[1:]
	}
	f.mu.Unlock()
	_, _ = w.Write(readFixture(f.t, fixture))
}

func (f *fakeSlack) methods() []string {
	var out []string
	for _, c := range f.calls {
		out = append(out, c.method)
	}
	return out
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return b
}

type memStore struct{ value string }

func (m *memStore) Load() (string, error) { return m.value, nil }
func (m *memStore) Save(v string) error   { m.value = v; return nil }

func storedPair(t *testing.T, exp time.Time) *memStore {
	t.Helper()
	v, err := apptoken.Pair{Token: accessToken, RefreshToken: refreshToken, Exp: exp.Unix()}.Encode()
	require.NoError(t, err)
	return &memStore{value: v}
}

func newTestSession(api *client.AppsClient, store tokenStore) *session {
	return &session{api: api, store: store, now: func() time.Time { return now }}
}

func captureOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := output.Writer
	output.Writer = &buf
	t.Cleanup(func() { output.Writer = orig })
	return &buf
}

func TestGetPrintsTheManifestIndentedAndSorted(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{"apps.manifest.export": {"manifest-export.json"}})
	out := captureOutput(t)

	require.NoError(t, runGet(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", ""))

	assert.Equal(t, string(readFixture(t, "manifest.json")), out.String())
	require.Len(t, fake.calls, 1)
	assert.Equal(t, "Bearer "+accessToken, fake.calls[0].auth)
	assert.Equal(t, "A0TESTAPP01", fake.calls[0].form["app_id"])
}

func TestGetWritesAFile(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.export": {"manifest-export.json"}})
	captureOutput(t)
	path := filepath.Join(t.TempDir(), "app.json")

	require.NoError(t, runGet(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", path))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(readFixture(t, "manifest.json")), string(got))
}

func TestSessionRotatesANearlyExpiredTokenBeforeTheCall(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{
		"tooling.tokens.rotate": {"rotate-response.json"},
		"apps.manifest.export":  {"manifest-export.json"},
	})
	captureOutput(t)
	store := storedPair(t, now.Add(time.Minute))

	require.NoError(t, runGet(newTestSession(api, store), "A0TESTAPP01", ""))

	assert.Equal(t, []string{"tooling.tokens.rotate", "apps.manifest.export"}, fake.methods())
	assert.Empty(t, fake.calls[0].auth, "rotate takes no access token")
	assert.Equal(t, refreshToken, fake.calls[0].form["refresh_token"])
	assert.Equal(t, "Bearer xoxe.xoxp-1-ROTATED-access-canary", fake.calls[1].auth)

	saved, err := apptoken.Parse(store.value)
	require.NoError(t, err)
	assert.Equal(t, "xoxe-1-ROTATED-refresh-canary", saved.RefreshToken)
	assert.Equal(t, int64(1790043200), saved.Exp)
	assert.Equal(t, "T0TEST", saved.TeamID)
}

func TestSessionRotatesOnceWhenSlackRejectsTheToken(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{
		"apps.manifest.export":  {"token-expired-response.json", "manifest-export.json"},
		"tooling.tokens.rotate": {"rotate-response.json"},
	})
	captureOutput(t)

	require.NoError(t, runGet(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", ""))

	assert.Equal(t, []string{"apps.manifest.export", "tooling.tokens.rotate", "apps.manifest.export"}, fake.methods())
}

func TestSessionDoesNotRotateTwice(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{
		"tooling.tokens.rotate": {"rotate-response.json"},
		"apps.manifest.export":  {"token-expired-response.json"},
	})
	captureOutput(t)

	err := runGet(newTestSession(api, storedPair(t, now.Add(time.Minute))), "A0TESTAPP01", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_expired")
	assert.Contains(t, err.Error(), "set-credential --key app_config_token")
	assert.Equal(t, []string{"tooling.tokens.rotate", "apps.manifest.export"}, fake.methods())
}

func TestValidateListsManifestProblems(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{"apps.manifest.validate": {"invalid-manifest-response.json"}})
	captureOutput(t)

	err := runValidate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "testdata/manifest.yaml", "A0TESTAPP01")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "/settings/event_subscriptions/request_url: Event subscription requests must be sent to an https URL.")
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(fake.calls[0].form["manifest"]), &sent))
	var want map[string]any
	require.NoError(t, json.Unmarshal(readFixture(t, "manifest.json"), &want))
	assert.Equal(t, want, sent, "the YAML file is sent as the same manifest as its JSON twin")
	assert.Equal(t, "A0TESTAPP01", fake.calls[0].form["app_id"])
}

func TestDiffExitsZeroWhenTheyMatch(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.export": {"manifest-export.json"}})
	out := captureOutput(t)

	require.NoError(t, runDiff(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", "testdata/manifest.yaml"))
	assert.Contains(t, out.String(), "No differences")
}

func TestDiffExitsOneAndListsDifferences(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.export": {"manifest-export.json"}})
	out := captureOutput(t)

	err := runDiff(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", "testdata/manifest-more-scopes.json")

	var exit *exitError
	require.True(t, errors.As(err, &exit))
	assert.Equal(t, 1, exit.ExitCode())
	assert.True(t, exit.Silent())
	assert.Equal(t, `~ /oauth_config/scopes/bot: ["chat:write","channels:read"] -> ["chat:write","channels:read","channels:history"]
+ /settings/interactivity: {"is_enabled":true} (file only)
`, out.String())
}

func TestDiffIgnoresDefaultsTheFileLeavesOut(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.export": {"manifest-export-with-defaults.json"}})
	out := captureOutput(t)

	err := runDiff(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", "testdata/manifest.json")

	var exit *exitError
	require.True(t, errors.As(err, &exit))
	assert.Equal(t, 1, exit.ExitCode())
	assert.Equal(t, "- /settings/interactivity: {\"is_enabled\":true} (live only)\n", out.String(),
		"false defaults Slack adds on export are not differences; a live-only setting that is on still is")
}

func TestDiffExitsTwoWhenItCannotCompare(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{})
	captureOutput(t)

	err := runDiff(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", "testdata/missing.json")

	var exit *exitError
	require.True(t, errors.As(err, &exit))
	assert.Equal(t, 2, exit.ExitCode())
	assert.False(t, exit.Silent())
}

func TestCreateWritesCredentialsOnlyToTheFile(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.create": {"create-response.json"}})
	out := captureOutput(t)
	path := filepath.Join(t.TempDir(), "creds.json")

	require.NoError(t, runCreate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "testdata/manifest.json", path))

	assert.NotContains(t, out.String(), "CANARY")
	assert.Contains(t, out.String(), "Created app A0TESTAPP01")
	assert.Contains(t, out.String(), "https://slack.com/oauth/v2/authorize?client_id=1111.2222")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	var creds client.AppCredentials
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &creds))
	assert.Equal(t, "CANARY-signing-secret-77aa", creds.SigningSecret)
}

func TestCreateRefusesAnExistingCredentialsFileBeforeCreating(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{})
	captureOutput(t)
	path := filepath.Join(t.TempDir(), "creds.json")
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o600))

	err := runCreate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "testdata/manifest.json", path)

	require.Error(t, err)
	assert.Empty(t, fake.calls, "no app is created when its credentials could not be kept")
}

func TestCreateWithoutACredentialsFilePrintsNoSecrets(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.create": {"create-response.json"}})
	out := captureOutput(t)

	require.NoError(t, runCreate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "testdata/manifest.json", ""))
	assert.NotContains(t, out.String(), "CANARY")
}

func TestUpdateSaysWhenAReinstallIsNeeded(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{"apps.manifest.update": {"update-response.json"}})
	out := captureOutput(t)

	require.NoError(t, runUpdate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", "testdata/manifest-more-scopes.json"))

	assert.Contains(t, out.String(), "reinstall the app")
	assert.Equal(t, "A0TESTAPP01", fake.calls[0].form["app_id"])
}

func TestDeleteRefusesWithoutATerminalOrForce(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{})
	captureOutput(t)

	err := runDelete(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", &deleteOptions{stdin: strings.NewReader("y\n")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--force")
	assert.Empty(t, fake.calls)
}

func TestDeleteAsksAndRespectsNo(t *testing.T) {
	fake, api := newFakeSlack(t, map[string][]string{})
	out := captureOutput(t)

	require.NoError(t, runDelete(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01",
		&deleteOptions{stdin: strings.NewReader("n\n"), terminal: true}))

	assert.Contains(t, out.String(), "Cancelled.")
	assert.Empty(t, fake.calls)
}

func TestDeleteWithConfirmationOrForce(t *testing.T) {
	for name, opts := range map[string]*deleteOptions{
		"confirmed": {stdin: strings.NewReader("yes\n"), terminal: true},
		"force":     {force: true},
	} {
		t.Run(name, func(t *testing.T) {
			fake, api := newFakeSlack(t, map[string][]string{"apps.manifest.delete": {"ok-response.json"}})
			out := captureOutput(t)

			require.NoError(t, runDelete(newTestSession(api, storedPair(t, now.Add(time.Hour))), "A0TESTAPP01", opts))

			assert.Contains(t, out.String(), "Deleted app A0TESTAPP01")
			assert.Equal(t, "A0TESTAPP01", fake.calls[0].form["app_id"])
		})
	}
}

func TestCreateRemovesTheCredentialsFileWhenCreateFails(t *testing.T) {
	_, api := newFakeSlack(t, map[string][]string{"apps.manifest.create": {"invalid-manifest-response.json"}})
	captureOutput(t)
	path := filepath.Join(t.TempDir(), "creds.json")

	err := runCreate(newTestSession(api, storedPair(t, now.Add(time.Hour))), "testdata/manifest.json", path)

	require.Error(t, err)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "no empty credentials file is left behind")
}

func TestDiffInvocationErrorsExitTwo(t *testing.T) {
	for name, args := range map[string][]string{
		"no app id":    {"diff", "--file", "testdata/manifest.json"},
		"unknown flag": {"diff", "A0TESTAPP01", "--bogus"},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := NewCmd()
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()

			var exit *exitError
			require.True(t, errors.As(err, &exit), "got %v", err)
			assert.Equal(t, 2, exit.ExitCode())
		})
	}
}
