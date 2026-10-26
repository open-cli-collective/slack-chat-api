package watch

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

func fixture(t *testing.T, name string) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func parentWatch() *Watch {
	w := &Watch{Channel: "C0123ABCD", TS: "1790000000.000100", ThreadTS: "1790000000.000100"}
	w.SetMessages([]client.Message{{User: "U0111AAAA", Text: "can someone approve the plan?", TS: "1790000000.000100", ThreadTS: "1790000000.000100"}})
	return w
}

func TestApply_FollowsTheThreadAndIgnoresTheRest(t *testing.T) {
	w := parentWatch()

	for _, name := range []string{"unrelated-message.json", "reply-other-thread.json", "parent-replied.json"} {
		assert.False(t, w.Apply(fixture(t, name)), name)
	}
	assert.Len(t, w.Messages, 1)

	require.True(t, w.Apply(fixture(t, "reply.json")))
	require.Len(t, w.Messages, 2)
	assert.Equal(t, "looks good to me", w.Messages[1].Text)

	require.True(t, w.Apply(fixture(t, "reaction-added.json")))
	assert.Equal(t, []client.Reaction{{Name: "white_check_mark", Count: 1, Users: []string{"U0333CCCC"}}}, w.Messages[0].Reactions)
	assert.False(t, w.Apply(fixture(t, "reaction-added.json")), "the same reaction twice is one reaction")

	require.True(t, w.Apply(fixture(t, "reply-edited.json")))
	assert.Equal(t, "looks good to me, ship it", w.Messages[1].Text)
	assert.NotNil(t, w.Messages[1].Edited)

	require.True(t, w.Apply(fixture(t, "reaction-removed.json")))
	assert.Empty(t, w.Messages[0].Reactions)

	require.True(t, w.Apply(fixture(t, "reply-deleted.json")))
	assert.Len(t, w.Messages, 1)
}

func TestApply_AWatchedReplyFollowsOnlyLaterMessages(t *testing.T) {
	// Watching the reply: the parent's reactions and earlier replies are not
	// part of it.
	w := &Watch{Channel: "C0123ABCD", TS: "1790000100.000200", ThreadTS: "1790000000.000100"}
	w.SetMessages([]client.Message{
		{User: "U0111AAAA", Text: "parent", TS: "1790000000.000100", ThreadTS: "1790000000.000100"},
		{User: "U0222BBBB", Text: "looks good to me", TS: "1790000100.000200", ThreadTS: "1790000000.000100"},
	})
	require.Len(t, w.Messages, 1)
	assert.False(t, w.Apply(fixture(t, "reaction-added.json")), "a reaction on the parent")
	assert.True(t, w.Apply(fixture(t, "reply-edited.json")))
}

func TestStore_RoundTripsAndKeepsRemovedWatchesRemoved(t *testing.T) {
	s, err := OpenDir(t.TempDir())
	require.NoError(t, err)
	w := parentWatch()
	w.AddedAt = time.Unix(1790000000, 0)
	w.Activity = "[2026-09-21 10:13] Ana: can someone approve the plan?\n"
	require.NoError(t, s.Save(w))

	got, err := s.Load(w.Ref())
	require.NoError(t, err)
	assert.Equal(t, w.Activity, got.Activity)
	all, err := s.List()
	require.NoError(t, err)
	assert.Len(t, all, 1)

	require.NoError(t, s.SaveListener(Listener{PID: 42, LastOK: time.Now()}))
	all, err = s.List()
	require.NoError(t, err)
	assert.Len(t, all, 1, "the listener file is not a watch")

	require.NoError(t, s.Remove(w.Ref()))
	require.NoError(t, s.Update(w), "updating a removed watch is not an error")
	_, err = s.Load(w.Ref())
	assert.ErrorIs(t, err, ErrNotWatched, "and does not bring it back")
	assert.ErrorIs(t, s.Remove(w.Ref()), ErrNotWatched)
}

func TestListener_LiveOnlyWhileItConfirmsTheConnection(t *testing.T) {
	now := time.Now()
	assert.False(t, Listener{}.Live(now))
	assert.True(t, Listener{LastOK: now.Add(-PingEvery)}.Live(now))
	assert.False(t, Listener{LastOK: now.Add(-LiveWithin)}.Live(now))
	assert.Contains(t, Listener{}.Describe(now), "slck watch listen")
}

type recorder struct {
	mu        sync.Mutex
	connected int
	alive     int
	// aliveBeforeConnected records Alive calls seen when Connected ran.
	aliveBeforeConnected int
	events               []json.RawMessage
}

func (r *recorder) Connected() {
	r.mu.Lock()
	r.connected++
	r.aliveBeforeConnected = r.alive
	r.mu.Unlock()
}
func (r *recorder) Alive() { r.mu.Lock(); r.alive++; r.mu.Unlock() }
func (r *recorder) Event(ev json.RawMessage) {
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
}

func TestListen_AcknowledgesEnvelopesDeliversEventsAndReconnectsWhenAsked(t *testing.T) {
	frames, err := os.Open(filepath.Join("testdata", "socket-frames.jsonl"))
	require.NoError(t, err)
	defer func() { _ = frames.Close() }()
	var script []string
	scanner := bufio.NewScanner(frames)
	for scanner.Scan() {
		script = append(script, scanner.Text())
	}

	acks := make(chan string, 4)
	var sessions int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		mu.Lock()
		sessions++
		first := sessions == 1
		mu.Unlock()
		if !first {
			// The reconnect: hold it open until the test ends.
			_, _, _ = conn.Read(r.Context())
			return
		}
		ctx := r.Context()
		go func() {
			for {
				_, data, err := conn.Read(ctx)
				if err != nil {
					return
				}
				acks <- string(data)
			}
		}()
		for _, frame := range script {
			if err := conn.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
				return
			}
		}
		<-ctx.Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opens := make(chan struct{}, 4)
	open := func(context.Context) (string, error) {
		opens <- struct{}{}
		return "ws" + strings.TrimPrefix(srv.URL, "http"), nil
	}
	rec := &recorder{}
	done := make(chan error, 1)
	go func() { done <- Listen(ctx, open, rec, func(string, ...any) {}) }()

	select {
	case ack := <-acks:
		assert.JSONEq(t, `{"envelope_id":"env-1"}`, ack)
	case <-time.After(5 * time.Second):
		t.Fatal("no acknowledgement")
	}
	for i := 0; i < 2; i++ {
		select {
		case <-opens:
		case <-time.After(5 * time.Second):
			t.Fatal("no reconnect after the disconnect frame")
		}
	}
	cancel()
	require.NoError(t, <-done)

	rec.mu.Lock()
	defer rec.mu.Unlock()
	assert.Equal(t, 1, rec.connected)
	assert.Zero(t, rec.aliveBeforeConnected, "Connected must see how long the listener was away")
	require.Len(t, rec.events, 1)
	assert.Contains(t, string(rec.events[0]), `"reaction":"eyes"`)
}
