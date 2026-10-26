package watch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/slack-chat-api/internal/watch"
)

const ref = "C0123ABCD/1790000000.000100"

func storeWith(t *testing.T, live bool) (*watch.Store, *watch.Watch) {
	t.Helper()
	s, err := watch.OpenDir(t.TempDir())
	require.NoError(t, err)
	w := &watch.Watch{Channel: "C0123ABCD", TS: "1790000000.000100", ThreadTS: "1790000000.000100",
		ExpiresAt: time.Now().Add(time.Hour), Activity: "before\n"}
	require.NoError(t, s.Save(w))
	l := watch.Listener{PID: 1}
	if live {
		l.LastOK = time.Now()
	}
	require.NoError(t, s.SaveListener(l))
	return s, w
}

func TestWait_ReturnsWhenTheThreadChanges(t *testing.T) {
	s, w := storeWith(t, true)
	go func() {
		time.Sleep(50 * time.Millisecond)
		w.Activity = "before\nafter\n"
		_ = s.Save(w)
	}()
	require.NoError(t, runWait(context.Background(), ref, 5*time.Second, 10*time.Millisecond, s, time.Now))
}

func TestWait_FailsInsteadOfWaitingOnAListenerThatIsGone(t *testing.T) {
	s, _ := storeWith(t, false)
	err := runWait(context.Background(), ref, 0, 10*time.Millisecond, s, time.Now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "slck watch listen")

	s, _ = storeWith(t, true)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.SaveListener(watch.Listener{PID: 1})
	}()
	err = runWait(context.Background(), ref, 0, 10*time.Millisecond, s, time.Now)
	require.Error(t, err, "the listener stopping ends the wait")
}

func TestWait_GivesUpAtTheTimeoutOrWhenTheWatchIsRemoved(t *testing.T) {
	s, _ := storeWith(t, true)
	err := runWait(context.Background(), ref, 50*time.Millisecond, 10*time.Millisecond, s, time.Now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no new activity")

	s, w := storeWith(t, true)
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Remove(w.Ref())
	}()
	err = runWait(context.Background(), ref, 0, 10*time.Millisecond, s, time.Now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not watched")
}
