// Package watch keeps the messages `slck watch` follows and applies Slack
// events to them. One listener process (`slck watch listen`) holds a Socket
// Mode connection and writes each watched message's thread, reactions
// included, to a file; `slck watch status` and `slck watch wait` read those
// files and never call Slack.
package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/open-cli-collective/cli-common/statedir"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/messageref"
)

// LiveWithin is how recently the listener must have confirmed its connection
// for status and wait to trust the watch files. The listener confirms every
// PingEvery, so this allows for a missed ping and a reconnect.
const LiveWithin = 75 * time.Second

// ErrNotWatched is returned for a ref that has no watch.
var ErrNotWatched = errors.New("not watched")

// Watch is one followed message: its thread from that message on, and the
// rendered text status and wait print.
type Watch struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
	// ThreadTS is the thread the message belongs to: its own ts for a
	// top-level message, the parent's for a reply.
	ThreadTS  string           `json:"thread_ts"`
	AddedAt   time.Time        `json:"added_at"`
	ExpiresAt time.Time        `json:"expires_at"`
	Messages  []client.Message `json:"messages"`
	// Activity is Messages rendered as `slck messages read` prints them. It
	// changes only when the thread does.
	Activity string `json:"activity"`
}

// Ref is the watched message's ref.
func (w *Watch) Ref() messageref.Ref {
	return messageref.Ref{ChannelID: w.Channel, TS: w.TS}
}

// SetMessages keeps the messages of the thread from the watched one on, in
// order.
func (w *Watch) SetMessages(messages []client.Message) {
	kept := make([]client.Message, 0, len(messages))
	for _, m := range messages {
		if m.TS >= w.TS {
			kept = append(kept, m)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].TS < kept[j].TS })
	w.Messages = kept
}

// Listener is the state the listener process publishes.
type Listener struct {
	PID            int       `json:"pid"`
	ConnectedSince time.Time `json:"connected_since,omitempty"`
	// LastOK is when the connection last answered a ping. A stopped
	// listener clears it.
	LastOK time.Time `json:"last_ok,omitempty"`
}

// Live reports whether the listener confirmed its connection recently.
func (l Listener) Live(now time.Time) bool {
	return !l.LastOK.IsZero() && now.Sub(l.LastOK) < LiveWithin
}

// Describe says whether the listener is running, for people.
func (l Listener) Describe(now time.Time) string {
	if l.Live(now) {
		return fmt.Sprintf("connected since %s (pid %d)", l.ConnectedSince.Local().Format("2006-01-02 15:04"), l.PID)
	}
	if l.LastOK.IsZero() {
		return "not running; start it with `slck watch listen`"
	}
	return fmt.Sprintf("not connected since %s; start it with `slck watch listen`", l.LastOK.Local().Format("2006-01-02 15:04"))
}

// Store is the directory of watch files.
type Store struct {
	dir string
}

// Open returns the store in slck's data directory, creating it.
func Open() (*Store, error) {
	base, err := statedir.Data{Tool: "slck"}.DataDirEnsured()
	if err != nil {
		return nil, err
	}
	return OpenDir(filepath.Join(base, "watches"))
}

// OpenDir returns the store in dir, creating it.
func OpenDir(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating watch directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

const listenerFile = "listener.json"

func (s *Store) path(ref messageref.Ref) string {
	return filepath.Join(s.dir, ref.ChannelID+"_"+ref.TS+".json")
}

// Load reads one watch. ErrNotWatched when there is none.
func (s *Store) Load(ref messageref.Ref) (*Watch, error) {
	var w Watch
	if err := readJSON(s.path(ref), &w); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", ref, ErrNotWatched)
		}
		return nil, err
	}
	return &w, nil
}

// List reads every watch, oldest first.
func (s *Store) List() ([]*Watch, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var watches []*Watch
	for _, e := range entries {
		if e.IsDir() || e.Name() == listenerFile || !strings.HasSuffix(e.Name(), ".json") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		var w Watch
		if err := readJSON(filepath.Join(s.dir, e.Name()), &w); err != nil {
			// Removed between ReadDir and the read.
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		watches = append(watches, &w)
	}
	sort.Slice(watches, func(i, j int) bool { return watches[i].AddedAt.Before(watches[j].AddedAt) })
	return watches, nil
}

// Save writes a watch, replacing any watch of the same message.
func (s *Store) Save(w *Watch) error {
	return s.withLock(func() error { return s.write(w) })
}

// Remove deletes a watch. ErrNotWatched when there is none.
func (s *Store) Remove(ref messageref.Ref) error {
	return s.withLock(func() error {
		err := os.Remove(s.path(ref))
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: %w", ref, ErrNotWatched)
		}
		return err
	})
}

// Modify changes one watch under the store lock and writes it when change
// reports a change. A watch removed meanwhile stays removed:
// ErrNotWatched.
func (s *Store) Modify(ref messageref.Ref, change func(*Watch) bool) error {
	return s.withLock(func() error {
		w, err := s.Load(ref)
		if err != nil {
			return err
		}
		if !change(w) {
			return nil
		}
		return s.write(w)
	})
}

// ModifyAll offers every watch to change under the store lock and writes
// the ones it changed. It returns the changed watches.
func (s *Store) ModifyAll(change func(*Watch) bool) ([]*Watch, error) {
	var changed []*Watch
	err := s.withLock(func() error {
		watches, err := s.List()
		if err != nil {
			return err
		}
		for _, w := range watches {
			if !change(w) {
				continue
			}
			if err := s.write(w); err != nil {
				return err
			}
			changed = append(changed, w)
		}
		return nil
	})
	return changed, err
}

func (s *Store) write(w *Watch) error {
	return writeJSON(s.path(w.Ref()), w)
}

const storeLock = ".lock"

// withLock runs fn holding the store's lock file, which serializes writers
// across processes: the listener, and add and remove from the CLI. Holders
// only read and write files; nothing that waits on the network runs under
// it.
func (s *Store) withLock(fn func() error) error {
	unlock, err := lockFile(filepath.Join(s.dir, storeLock))
	if err != nil {
		return fmt.Errorf("locking the watch store: %w", err)
	}
	defer unlock()
	return fn()
}

// Listener reads the listener's state; a listener that never ran has none.
func (s *Store) Listener() (Listener, error) {
	var l Listener
	err := readJSON(filepath.Join(s.dir, listenerFile), &l)
	if errors.Is(err, os.ErrNotExist) {
		return Listener{}, nil
	}
	return l, err
}

// SaveListener publishes the listener's state.
func (s *Store) SaveListener(l Listener) error {
	return writeJSON(filepath.Join(s.dir, listenerFile), l)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s does not parse: %w", path, err)
	}
	return nil
}

// writeJSON replaces path in one rename, so a reader never sees half a file.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
