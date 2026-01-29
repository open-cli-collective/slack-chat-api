package client

import (
	"fmt"
	"strings"
)

// ResolveChannel takes a channel identifier (ID or name) and returns the channel ID.
// If the input looks like a channel ID (starts with C, G, or D), it's returned as-is.
// Otherwise, it's treated as a channel name and looked up via the Slack API.
func (c *Client) ResolveChannel(channel string) (string, error) {
	// Strip leading # if present (common user mistake)
	channel = strings.TrimPrefix(channel, "#")

	if channel == "" {
		return "", fmt.Errorf("channel cannot be empty")
	}

	// If it looks like a channel ID, return it as-is
	if IsChannelID(channel) {
		return channel, nil
	}

	// Otherwise, look it up by name
	return c.lookupChannelByName(channel)
}

// IsChannelID returns true if the string looks like a Slack channel ID.
// Channel IDs start with:
//   - C = public channel
//   - G = private channel (group)
//   - D = direct message
//
// Channel IDs are typically 9-11 characters and contain a mix of uppercase letters
// and numbers (e.g., C02DF3BEUGN). Pure alphabetic strings like "GENERAL" are
// treated as channel names, not IDs.
func IsChannelID(s string) bool {
	if len(s) < 2 {
		return false
	}
	prefix := s[0]
	if prefix != 'C' && prefix != 'G' && prefix != 'D' {
		return false
	}
	rest := s[1:]
	if len(rest) == 0 {
		return false
	}

	// Check that the rest is alphanumeric (uppercase) AND contains at least one digit.
	// This distinguishes channel IDs (C02DF3BEUGN) from words (GENERAL).
	hasDigit := false
	for _, c := range rest {
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c < 'A' || c > 'Z' {
			return false // not uppercase letter or digit
		}
	}
	return hasDigit
}

// lookupChannelByName searches for a channel by name and returns its ID.
func (c *Client) lookupChannelByName(name string) (string, error) {
	// Normalize the name (lowercase, strip #)
	name = strings.ToLower(strings.TrimPrefix(name, "#"))

	// Search across all channel types (public, private, mpim, im)
	channels, err := c.ListChannels("public_channel,private_channel", false, 1000)
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}

	for _, ch := range channels {
		if strings.EqualFold(ch.Name, name) {
			return ch.ID, nil
		}
	}

	return "", fmt.Errorf("channel '%s' not found. Use 'slck channels list' to see available channels", name)
}
