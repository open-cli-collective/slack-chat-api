package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// PingEvery is how often the listener confirms its connection is alive.
const PingEvery = 20 * time.Second

const (
	pingTimeout = 10 * time.Second
	maxBackoff  = 30 * time.Second
	// Events carry whole messages with their blocks and files.
	readLimit = 4 << 20
)

// Handler receives what a Socket Mode connection delivers.
type Handler interface {
	// Connected is called when Slack says hello on a new connection, before
	// any Alive on it, so the handler can tell how long it was away.
	Connected()
	// Event is called with the inner event of each Events API envelope.
	Event(event json.RawMessage)
	// Alive is called whenever the connection answers a ping.
	Alive()
}

// OpenFunc asks Slack for a Socket Mode URL (apps.connections.open).
type OpenFunc func(ctx context.Context) (string, error)

// Logf reports connection changes.
type Logf func(format string, args ...any)

type envelope struct {
	Type       string          `json:"type"`
	EnvelopeID string          `json:"envelope_id"`
	Reason     string          `json:"reason"`
	Payload    json.RawMessage `json:"payload"`
}

// Listen keeps a Socket Mode connection open until ctx is done, reconnecting
// with backoff when it drops and when Slack asks for a new one.
func Listen(ctx context.Context, open OpenFunc, h Handler, logf Logf) error {
	backoff := time.Second
	for {
		helloed, err := session(ctx, open, h, logf)
		if ctx.Err() != nil {
			return nil
		}
		if helloed {
			backoff = time.Second
		}
		if err != nil {
			logf("connection lost: %v; reconnecting in %s", err, backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

// session runs one connection. It returns nil when Slack asked the client
// to reconnect, and whether Slack said hello on it.
func session(ctx context.Context, open OpenFunc, h Handler, logf Logf) (helloed bool, err error) {
	url, err := open(ctx)
	if err != nil {
		return false, err
	}
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(readLimit)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		ticker := time.NewTicker(PingEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pctx, pcancel := context.WithTimeout(ctx, pingTimeout)
				err := conn.Ping(pctx)
				pcancel()
				if err != nil {
					// Closing makes the read below fail, which reconnects.
					_ = conn.CloseNow()
					return
				}
				h.Alive()
			}
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return helloed, err
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			logf("skipping a frame that does not parse: %v", err)
			continue
		}
		if env.EnvelopeID != "" {
			ack, _ := json.Marshal(map[string]string{"envelope_id": env.EnvelopeID})
			if err := conn.Write(ctx, websocket.MessageText, ack); err != nil {
				return helloed, fmt.Errorf("acknowledging an envelope: %w", err)
			}
		}
		switch env.Type {
		case "hello":
			helloed = true
			h.Connected()
		case "disconnect":
			logf("Slack asked for a new connection (%s)", env.Reason)
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return helloed, nil
		case "events_api":
			var p struct {
				Event json.RawMessage `json:"event"`
			}
			if err := json.Unmarshal(env.Payload, &p); err != nil || len(p.Event) == 0 {
				continue
			}
			h.Event(p.Event)
		}
	}
}
