package client

import (
	"encoding/json"
	"net/url"
)

// PinnedItem is one entry of pins.list. Type is "message" for a pinned
// message; Slack can also return "file" and "file_comment" items, which carry
// File instead of Message.
type PinnedItem struct {
	Type      string   `json:"type"`
	Channel   string   `json:"channel"`
	Created   int64    `json:"created"`
	CreatedBy string   `json:"created_by"`
	Message   *Message `json:"message,omitempty"`
	File      *File    `json:"file,omitempty"`
}

// AddPin pins a message to a channel. It reports whether the message was
// already pinned, which Slack signals with an already_pinned error.
func (c *Client) AddPin(channel, timestamp string) (alreadyPinned bool, err error) {
	_, err = c.post("pins.add", map[string]interface{}{
		"channel":   channel,
		"timestamp": timestamp,
	})
	if IsSlackError(err, "already_pinned") {
		return true, nil
	}
	return false, err
}

// RemovePin unpins a message from a channel. It reports whether the message
// was not pinned, which Slack signals with a no_pin or not_pinned error.
func (c *Client) RemovePin(channel, timestamp string) (notPinned bool, err error) {
	_, err = c.post("pins.remove", map[string]interface{}{
		"channel":   channel,
		"timestamp": timestamp,
	})
	if IsSlackError(err, "no_pin") || IsSlackError(err, "not_pinned") {
		return true, nil
	}
	return false, err
}

// ListPins returns the items pinned to a channel.
func (c *Client) ListPins(channel string) ([]PinnedItem, error) {
	params := url.Values{}
	params.Set("channel", channel)

	body, err := c.get("pins.list", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []PinnedItem `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}
