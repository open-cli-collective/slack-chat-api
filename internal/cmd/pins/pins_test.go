package pins

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/render"
)

const (
	testChannel = "C1234567890"
	testTS      = "1234567890.123456"
)

// slackCall is one request the fake Slack API received.
type slackCall struct {
	method string
	path   string
	query  map[string]string
	body   map[string]interface{}
}

// fakeSlack answers each API method with a testdata fixture and records the
// requests. users.info answers with users-info-<user>.json.
func fakeSlack(t *testing.T, fixtures map[string]string) (*client.Client, *[]slackCall) {
	t.Helper()
	var calls []slackCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := slackCall{method: r.Method, path: r.URL.Path, query: map[string]string{}}
		for k := range r.URL.Query() {
			call.query[k] = r.URL.Query().Get(k)
		}
		if r.Method == http.MethodPost {
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&call.body))
		}
		calls = append(calls, call)

		fixture, ok := fixtures[r.URL.Path]
		if r.URL.Path == "/users.info" {
			fixture, ok = "users-info-"+call.query["user"]+".json", true
		}
		if !assert.True(t, ok, "unexpected call to %s", r.URL.Path) {
			http.Error(w, "unexpected call", http.StatusInternalServerError)
			return
		}
		b, err := os.ReadFile(filepath.Join("testdata", fixture))
		if !assert.NoError(t, err) {
			http.Error(w, "missing fixture", http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(server.Close)
	return client.NewWithConfig(server.URL, "test-token", nil), &calls
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf strings.Builder
	orig := output.Writer
	output.Writer = &buf
	t.Cleanup(func() { output.Writer = orig })
	fn()
	return buf.String()
}

func TestRunAdd(t *testing.T) {
	for name, tc := range map[string]struct {
		fixture string
		want    string
	}{
		"pins":           {"ok.json", "Pinned message 1234567890.123456 in C1234567890\n"},
		"already pinned": {"already-pinned.json", "Message 1234567890.123456 is already pinned in C1234567890\n"},
	} {
		t.Run(name, func(t *testing.T) {
			c, calls := fakeSlack(t, map[string]string{"/pins.add": tc.fixture})

			out := captureOutput(t, func() {
				require.NoError(t, runAdd(testChannel, testTS, c))
			})

			assert.Equal(t, tc.want, out)
			require.Len(t, *calls, 1)
			call := (*calls)[0]
			assert.Equal(t, http.MethodPost, call.method)
			assert.Equal(t, "/pins.add", call.path)
			assert.Equal(t, map[string]interface{}{"channel": testChannel, "timestamp": testTS}, call.body)
		})
	}
}

func TestRunAdd_NormalizesTimestamp(t *testing.T) {
	for _, input := range []string{
		"p1234567890123456",
		"https://example.slack.com/archives/C1234567890/p1234567890123456",
	} {
		t.Run(input, func(t *testing.T) {
			c, calls := fakeSlack(t, map[string]string{"/pins.add": "ok.json"})

			captureOutput(t, func() {
				require.NoError(t, runAdd(testChannel, input, c))
			})

			require.Len(t, *calls, 1)
			assert.Equal(t, testTS, (*calls)[0].body["timestamp"])
		})
	}
}

func TestRunAdd_InvalidTimestamp(t *testing.T) {
	c, calls := fakeSlack(t, nil)

	err := runAdd(testChannel, "not-a-timestamp", c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
	assert.Empty(t, *calls)
}

func TestRunRemove(t *testing.T) {
	for name, tc := range map[string]struct {
		fixture string
		want    string
	}{
		"unpins":                 {"ok.json", "Unpinned message 1234567890.123456 in C1234567890\n"},
		"not pinned, no_pin":     {"no-pin.json", "Message 1234567890.123456 is not pinned in C1234567890\n"},
		"not pinned, not_pinned": {"not-pinned.json", "Message 1234567890.123456 is not pinned in C1234567890\n"},
	} {
		t.Run(name, func(t *testing.T) {
			c, calls := fakeSlack(t, map[string]string{"/pins.remove": tc.fixture})

			out := captureOutput(t, func() {
				require.NoError(t, runRemove(testChannel, testTS, c))
			})

			assert.Equal(t, tc.want, out)
			require.Len(t, *calls, 1)
			call := (*calls)[0]
			assert.Equal(t, http.MethodPost, call.method)
			assert.Equal(t, "/pins.remove", call.path)
			assert.Equal(t, map[string]interface{}{"channel": testChannel, "timestamp": testTS}, call.body)
		})
	}
}

func TestRunRemove_InvalidTimestamp(t *testing.T) {
	c, calls := fakeSlack(t, nil)

	err := runRemove(testChannel, "not-a-timestamp", c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
	assert.Empty(t, *calls)
}

func TestRunList(t *testing.T) {
	c, calls := fakeSlack(t, map[string]string{"/pins.list": "pins-list.json"})

	out := captureOutput(t, func() {
		require.NoError(t, runList(testChannel, c))
	})

	want := "TS | PINNED | PINNED_BY | AUTHOR | TEXT\n" +
		"1699999000.000200 | " + render.Timestamp("1700000200") + " | bob | alice | On call this week: @bob\n" +
		"- | " + render.Timestamp("1700000100") + " | bob | - | [file] Runbook\n" +
		"1699990000.000100 | " + render.Timestamp("1700000000") + " | alice | bob | Deploy checklist Step one ¦ step two\n"
	assert.Equal(t, want, out)
	require.NotEmpty(t, *calls)
	assert.Equal(t, http.MethodGet, (*calls)[0].method)
	assert.Equal(t, "/pins.list", (*calls)[0].path)
	assert.Equal(t, testChannel, (*calls)[0].query["channel"])
}

func TestRunList_Empty(t *testing.T) {
	c, _ := fakeSlack(t, map[string]string{"/pins.list": "pins-list-empty.json"})

	out := captureOutput(t, func() {
		require.NoError(t, runList(testChannel, c))
	})

	assert.Equal(t, "No pins in C1234567890\n", out)
}

func TestMissingScopeNamesTheScope(t *testing.T) {
	for name, tc := range map[string]struct {
		path  string
		run   func(*client.Client) error
		scope string
	}{
		"add":    {"/pins.add", func(c *client.Client) error { return runAdd(testChannel, testTS, c) }, "pins:write"},
		"remove": {"/pins.remove", func(c *client.Client) error { return runRemove(testChannel, testTS, c) }, "pins:write"},
		"list":   {"/pins.list", func(c *client.Client) error { return runList(testChannel, c) }, "pins:read"},
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := fakeSlack(t, map[string]string{tc.path: "missing-scope.json"})

			err := tc.run(c)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing_scope")
			assert.Contains(t, err.Error(), "The token is missing the "+tc.scope+" scope. Add "+tc.scope+" to the app's")
		})
	}
}

func TestOtherErrorsKeepTheirHint(t *testing.T) {
	c, _ := fakeSlack(t, map[string]string{"/pins.add": "channel-not-found.json"})

	err := runAdd(testChannel, testTS, c)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pin message: slack API error: channel_not_found")
	assert.Contains(t, err.Error(), "Verify the channel ID is correct")
	assert.NotContains(t, err.Error(), "pins:write")
}

func TestRuntimeErrorsPrintNoUsage(t *testing.T) {
	cmd := NewCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"add", testChannel, "not-a-timestamp"})

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
	assert.Empty(t, out.String())
}
