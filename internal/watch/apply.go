package watch

import (
	"encoding/json"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

// event is the part of an Events API event that decides which watch it
// touches. Slack sends message and reaction events for every conversation
// the user is in; most touch no watch.
type event struct {
	Type            string          `json:"type"`
	Subtype         string          `json:"subtype"`
	Channel         string          `json:"channel"`
	DeletedTS       string          `json:"deleted_ts"`
	Message         *client.Message `json:"message"`
	PreviousMessage *client.Message `json:"previous_message"`
	User            string          `json:"user"`
	Reaction        string          `json:"reaction"`
	Item            struct {
		Type    string `json:"type"`
		Channel string `json:"channel"`
		TS      string `json:"ts"`
	} `json:"item"`
}

// Apply changes the watch for one event and reports whether it changed.
func (w *Watch) Apply(raw json.RawMessage) bool {
	var ev event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return false
	}
	switch ev.Type {
	case "message":
		if ev.Channel != w.Channel {
			return false
		}
		switch ev.Subtype {
		case "message_changed":
			if ev.Message == nil || !w.inThread(*ev.Message) {
				return false
			}
			edited := *ev.Message
			if old := w.find(edited.TS); old >= 0 && edited.Reactions == nil {
				edited.Reactions = w.Messages[old].Reactions
			}
			return w.upsert(edited)
		case "message_deleted":
			return w.remove(ev.DeletedTS)
		case "message_replied":
			// The parent's reply count; the reply arrives as its own event.
			return false
		default:
			var m client.Message
			if err := json.Unmarshal(raw, &m); err != nil || !w.inThread(m) {
				return false
			}
			return w.upsert(m)
		}
	case "reaction_added", "reaction_removed":
		if ev.Item.Type != "message" || ev.Item.Channel != w.Channel {
			return false
		}
		i := w.find(ev.Item.TS)
		if i < 0 {
			return false
		}
		return react(&w.Messages[i], ev.Reaction, ev.User, ev.Type == "reaction_added")
	}
	return false
}

func (w *Watch) inThread(m client.Message) bool {
	if m.TS < w.TS {
		return false
	}
	return m.TS == w.ThreadTS || m.ThreadTS == w.ThreadTS
}

func (w *Watch) find(ts string) int {
	for i, m := range w.Messages {
		if m.TS == ts {
			return i
		}
	}
	return -1
}

func (w *Watch) upsert(m client.Message) bool {
	if i := w.find(m.TS); i >= 0 {
		w.Messages[i] = m
		return true
	}
	w.SetMessages(append(w.Messages, m))
	return true
}

func (w *Watch) remove(ts string) bool {
	i := w.find(ts)
	if i < 0 {
		return false
	}
	w.Messages = append(w.Messages[:i], w.Messages[i+1:]...)
	return true
}

func react(m *client.Message, name, user string, added bool) bool {
	for i := range m.Reactions {
		r := &m.Reactions[i]
		if r.Name != name {
			continue
		}
		has := -1
		for j, u := range r.Users {
			if u == user {
				has = j
			}
		}
		switch {
		case added && has < 0:
			r.Users = append(r.Users, user)
			r.Count++
		case !added && has >= 0:
			r.Users = append(r.Users[:has], r.Users[has+1:]...)
			r.Count--
			if r.Count <= 0 {
				m.Reactions = append(m.Reactions[:i], m.Reactions[i+1:]...)
			}
		default:
			return false
		}
		return true
	}
	if !added {
		return false
	}
	m.Reactions = append(m.Reactions, client.Reaction{Name: name, Count: 1, Users: []string{user}})
	return true
}
