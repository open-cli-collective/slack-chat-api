package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/messageref"
)

// ThreadLimit bounds the replies read when a watch starts or catches up.
const ThreadLimit = 1000

// CatchUpAfter is how long the connection may be gone before the listener
// rereads every watched thread on reconnecting. A reconnect Slack asks for
// opens the new connection within a second or two and misses nothing.
const CatchUpAfter = 15 * time.Second

// ThreadReader reads a thread from Slack.
type ThreadReader interface {
	GetThreadReplies(channel, threadTS string, limit int, oldest string) ([]client.Message, error)
}

// Render turns a watch's messages into the text status and wait compare.
type Render func([]client.Message) string

// Syncer keeps the watch files current: it starts watches, applies the
// events a Socket Mode connection delivers, and catches up after the
// connection was away. Every change to a watch's messages goes through it,
// so Activity always matches Messages.
type Syncer struct {
	store   *Store
	threads ThreadReader
	render  Render
	now     func() time.Time
	logf    Logf

	// mu guards state and the catch-up bookkeeping. Alive runs on the
	// connection's ping goroutine, Connected and Event on its read loop.
	mu    sync.Mutex
	state Listener
	// catchingUp is set while a catch-up runs; events that arrive meanwhile
	// are kept in during, and applied again to each thread it rereads.
	catchingUp bool
	during     []json.RawMessage
	ctx        context.Context
	wg         sync.WaitGroup
}

// NewSyncer returns a syncer for the process pid. ctx bounds catch-ups.
func NewSyncer(ctx context.Context, store *Store, threads ThreadReader, render Render, now func() time.Time, pid int, logf Logf) *Syncer {
	return &Syncer{store: store, threads: threads, render: render, now: now, logf: logf, state: Listener{PID: pid}, ctx: ctx}
}

// Add starts following a message: it reads the thread once, so status shows
// it at once and wait sees only what comes after.
func (s *Syncer) Add(ref messageref.Ref, days int) (*Watch, error) {
	now := s.now()
	w := &Watch{Channel: ref.ChannelID, TS: ref.TS, AddedAt: now, ExpiresAt: now.AddDate(0, 0, days)}
	if err := s.refresh(w); err != nil {
		return nil, err
	}
	if err := s.store.Save(w); err != nil {
		return nil, err
	}
	return w, nil
}

// refresh reads the watched message's thread from Slack and renders it.
func (s *Syncer) refresh(w *Watch) error {
	thread := w.ThreadTS
	if thread == "" {
		thread = w.TS
	}
	msgs, err := s.threads.GetThreadReplies(w.Channel, thread, ThreadLimit, "")
	if err != nil {
		return err
	}
	if w.ThreadTS == "" {
		w.ThreadTS = w.TS
		for _, m := range msgs {
			if m.TS == w.TS && m.ThreadTS != "" {
				w.ThreadTS = m.ThreadTS
			}
		}
		if w.ThreadTS != w.TS {
			// A reply: its thread is read from the parent.
			if msgs, err = s.threads.GetThreadReplies(w.Channel, w.ThreadTS, ThreadLimit, ""); err != nil {
				return err
			}
		}
	}
	if len(msgs) == 0 {
		return fmt.Errorf("no message %s", w.Ref())
	}
	w.SetMessages(msgs)
	w.Activity = s.render(w.Messages)
	return nil
}

// Connected records a new connection and, when the listener was away long
// enough to miss events, rereads every watched thread in the background so
// the connection keeps reading meanwhile.
func (s *Syncer) Connected() {
	s.mu.Lock()
	now := s.now()
	gap := s.state.LastOK.IsZero() || now.Sub(s.state.LastOK) > CatchUpAfter
	if s.state.ConnectedSince.IsZero() || gap {
		s.state.ConnectedSince = now
	}
	s.saveState(now)
	start := gap && !s.catchingUp
	if start {
		s.catchingUp = true
		s.during = nil
	}
	s.mu.Unlock()
	s.logf("connected")
	if start {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.catchUp()
		}()
	}
}

// catchUp rereads every watched thread and drops expired watches.
func (s *Syncer) catchUp() {
	defer func() {
		s.mu.Lock()
		s.catchingUp = false
		s.during = nil
		s.mu.Unlock()
	}()
	watches, err := s.store.List()
	if err != nil {
		s.logf("listing watches: %v", err)
		return
	}
	for _, w := range watches {
		if s.ctx.Err() != nil {
			return
		}
		if s.now().After(w.ExpiresAt) {
			if err := s.store.Remove(w.Ref()); err == nil {
				s.logf("%s expired", w.Ref())
			}
			continue
		}
		fresh := *w
		if err := s.refresh(&fresh); err != nil {
			s.logf("catching up on %s: %v", w.Ref(), err)
			continue
		}
		s.mu.Lock()
		err := s.store.Modify(w.Ref(), func(cur *Watch) bool {
			before := cur.Activity
			cur.ThreadTS = fresh.ThreadTS
			cur.Messages = fresh.Messages
			// Events that arrived while the thread was being read may be
			// newer than what was read; applying one again is harmless.
			for _, ev := range s.during {
				cur.Apply(ev)
			}
			cur.Activity = s.render(cur.Messages)
			return cur.Activity != before
		})
		s.mu.Unlock()
		switch {
		case errors.Is(err, ErrNotWatched):
		case err != nil:
			s.logf("saving %s: %v", w.Ref(), err)
		}
	}
	s.logf("caught up on %d watches", len(watches))
}

// Event applies one event to every watch it touches.
func (s *Syncer) Event(ev json.RawMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.catchingUp {
		s.during = append(s.during, ev)
	}
	changed, err := s.store.ModifyAll(func(w *Watch) bool {
		if !w.Apply(ev) {
			return false
		}
		w.Activity = s.render(w.Messages)
		return true
	})
	if err != nil {
		s.logf("applying an event: %v", err)
	}
	for _, w := range changed {
		s.logf("%s: activity", w.Ref())
	}
}

// Alive records that the connection answered a ping.
func (s *Syncer) Alive() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveState(s.now())
}

func (s *Syncer) saveState(now time.Time) {
	s.state.LastOK = now
	if err := s.store.SaveListener(s.state); err != nil {
		s.logf("saving listener state: %v", err)
	}
}

// Close waits for a running catch-up and publishes that the listener
// stopped, so status and wait fail at once instead of after LiveWithin.
func (s *Syncer) Close() {
	s.wg.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.LastOK = time.Time{}
	_ = s.store.SaveListener(s.state)
}
