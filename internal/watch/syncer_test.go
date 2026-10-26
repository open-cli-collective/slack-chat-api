package watch

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/messageref"
)

// threads is a fake Slack: each thread's messages by thread ts.
type threads struct {
	mu    sync.Mutex
	byTS  map[string][]client.Message
	reads int
	// gate, when set, holds every read until it is closed.
	gate chan struct{}
}

func (f *threads) GetThreadReplies(channel, threadTS string, limit int, oldest string) ([]client.Message, error) {
	if f.gate != nil {
		<-f.gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return append([]client.Message(nil), f.byTS[threadTS]...), nil
}

func (f *threads) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) { c.mu.Lock(); c.t = c.t.Add(d); c.mu.Unlock() }

func renderTexts(msgs []client.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		b.WriteString(m.Text)
		for _, r := range m.Reactions {
			b.WriteString(" :" + r.Name + ":")
		}
		b.WriteString("\n")
	}
	return b.String()
}

const parentTS = "1790000000.000100"

var parentRef = messageref.Ref{ChannelID: "C0123ABCD", TS: parentTS}

func newSyncer(t *testing.T, f *threads) (*Syncer, *Store, *clock) {
	t.Helper()
	store, err := OpenDir(t.TempDir())
	require.NoError(t, err)
	c := &clock{t: time.Unix(1790000000, 0)}
	return NewSyncer(context.Background(), store, f, Renderers{Render: renderTexts, Cached: renderTexts}, c.now, 42, func(string, ...any) {}), store, c
}

func parentThread(texts ...string) []client.Message {
	msgs := []client.Message{{User: "U0111AAAA", Text: "can someone approve the plan?", TS: parentTS, ThreadTS: parentTS}}
	for i, text := range texts {
		msgs = append(msgs, client.Message{User: "U0222BBBB", Text: text, TS: "1790000100.00020" + string(rune('0'+i)), ThreadTS: parentTS})
	}
	return msgs
}

func activity(t *testing.T, store *Store) string {
	t.Helper()
	w, err := store.Load(parentRef)
	require.NoError(t, err)
	return w.Activity
}

func TestSyncer_CatchesUpOnFirstConnectAndAfterAGapOnly(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	s, store, c := newSyncer(t, f)
	_, err := s.Add(parentRef, 14)
	require.NoError(t, err)

	f.byTS[parentTS] = parentThread("approved")
	s.Connected()
	s.wg.Wait()
	assert.Contains(t, activity(t, store), "approved", "the first connection catches up")
	reads := f.readCount()

	s.Alive()
	c.add(CatchUpAfter / 2)
	s.Connected()
	s.wg.Wait()
	assert.Equal(t, reads, f.readCount(), "a quick reconnect does not reread")

	c.add(2 * CatchUpAfter)
	s.Connected()
	s.wg.Wait()
	assert.Greater(t, f.readCount(), reads, "a reconnect after a gap rereads")

	l, err := store.Listener()
	require.NoError(t, err)
	assert.Equal(t, 42, l.PID)
	s.Close()
	l, err = store.Listener()
	require.NoError(t, err)
	assert.False(t, l.Live(c.now()), "a closed listener is not live")
}

func TestSyncer_CatchUpDropsExpiredWatches(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	s, store, c := newSyncer(t, f)
	_, err := s.Add(parentRef, 1)
	require.NoError(t, err)
	c.add(48 * time.Hour)
	s.Connected()
	s.wg.Wait()
	_, err = store.Load(parentRef)
	assert.ErrorIs(t, err, ErrNotWatched)
}

func TestSyncer_KeepsAnEventThatArrivesWhileCatchingUp(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	s, store, _ := newSyncer(t, f)
	_, err := s.Add(parentRef, 14)
	require.NoError(t, err)

	// The catch-up read returns the thread as it was before the reaction.
	f.gate = make(chan struct{})
	s.Connected()
	s.Event(fixture(t, "reaction-added.json"))
	assert.Contains(t, activity(t, store), ":white_check_mark:", "applied at once")
	close(f.gate)
	s.wg.Wait()
	assert.Contains(t, activity(t, store), ":white_check_mark:", "and again on top of the catch-up read")
}

func TestSyncer_AddingAReplyFollowsItsThreadFromThatReplyOn(t *testing.T) {
	thread := parentThread("first reply", "second reply")
	f := &threads{byTS: map[string][]client.Message{parentTS: thread}}
	f.byTS[thread[1].TS] = thread[1:2] // Slack returns just the reply for a reply's ts
	s, _, _ := newSyncer(t, f)
	w, err := s.Add(messageref.Ref{ChannelID: "C0123ABCD", TS: thread[1].TS}, 14)
	require.NoError(t, err)
	assert.Equal(t, parentTS, w.ThreadTS)
	assert.Equal(t, "first reply\nsecond reply\n", w.Activity)
}

func TestSyncer_CallbacksFromBothGoroutinesAreSafe(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	s, store, _ := newSyncer(t, f)
	_, err := s.Add(parentRef, 14)
	require.NoError(t, err)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			s.Alive()
		}
	}()
	go func() {
		defer wg.Done()
		s.Connected()
		for i := 0; i < 50; i++ {
			s.Event(fixture(t, "reaction-added.json"))
			s.Event(fixture(t, "reaction-removed.json"))
		}
	}()
	wg.Wait()
	s.wg.Wait()
	assert.NotContains(t, activity(t, store), ":white_check_mark:")
}

func TestSyncer_AliveDoesNotWaitOnTheStore(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	s, store, _ := newSyncer(t, f)
	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = store.withLock(func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	defer close(release)
	done := make(chan struct{})
	go func() { s.Alive(); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Alive waited on the store lock")
	}
}

func TestSyncer_ResolvesNamesBeforeLockingTheStore(t *testing.T) {
	f := &threads{byTS: map[string][]client.Message{parentTS: parentThread()}}
	store, err := OpenDir(t.TempDir())
	require.NoError(t, err)
	c := &clock{t: time.Unix(1790000000, 0)}
	slow := make(chan struct{})
	var blocking sync.Once
	entered := make(chan struct{})
	renderers := Renderers{
		// The network render: the first call after Add blocks, as a slow
		// users.info would.
		Render: func(msgs []client.Message) string { return renderTexts(msgs) },
		Cached: renderTexts,
	}
	s := NewSyncer(context.Background(), store, f, renderers, c.now, 42, func(string, ...any) {})
	_, err = s.Add(parentRef, 14)
	require.NoError(t, err)
	s.render.Render = func(msgs []client.Message) string {
		blocking.Do(func() { close(entered); <-slow })
		return renderTexts(msgs)
	}

	evented := make(chan struct{})
	go func() { s.Event(fixture(t, "reaction-added.json")); close(evented) }()
	<-entered
	saved := make(chan error, 1)
	go func() {
		saved <- store.Save(&Watch{Channel: "C0123ABCD", TS: "1790000900.000100", ThreadTS: "1790000900.000100"})
	}()
	select {
	case err := <-saved:
		require.NoError(t, err, "the store is free while names are resolved")
	case <-time.After(time.Second):
		t.Fatal("a slow name lookup held the store lock")
	}
	close(slow)
	<-evented
	assert.Contains(t, activity(t, store), ":white_check_mark:")
}
