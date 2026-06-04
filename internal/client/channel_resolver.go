package client

import (
	"fmt"
	"strings"
)

// ResolveChannel takes a conversation identifier and returns the channel ID to
// act on. It accepts:
//   - channel IDs (C/G/D...)        returned as-is
//   - channel names ("general", "#general")  looked up via the Slack API
func (c *Client) ResolveChannel(channel string) (string, error) {
	if channel == "" {
		return "", fmt.Errorf("channel cannot be empty")
	}

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

// ResolveMessageDestination takes a message destination and returns a Slack
// conversation ID suitable for message APIs. User IDs and handles are resolved
// by opening a DM; channel-like inputs are delegated to ResolveChannel.
func (c *Client) ResolveMessageDestination(destination string) (string, error) {
	if destination == "" {
		return "", fmt.Errorf("channel cannot be empty")
	}

	// "@handle" -> resolve the username to a user ID, then open a DM.
	if handle := strings.TrimPrefix(destination, "@"); handle != destination {
		userID, err := c.resolveUserHandle(handle)
		if err != nil {
			return "", err
		}
		return c.OpenDM(userID)
	}

	// A bare user ID (U.../W...) -> open a DM and return its IM channel ID.
	// chat.postMessage rejects raw user IDs, so the DM must be opened first.
	if IsUserID(destination) {
		return c.OpenDM(destination)
	}

	return c.ResolveChannel(destination)
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
			return false // not an uppercase letter
		}
	}
	return hasDigit
}

// IsUserID returns true if the string looks like a Slack user ID.
// User IDs start with:
//   - U = regular user
//   - W = Enterprise Grid user
//
// Like channel IDs, they are uppercase alphanumeric and contain at least one
// digit, which distinguishes them from plain words (e.g. "U" or "Us").
func IsUserID(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != 'U' && s[0] != 'W' {
		return false
	}
	rest := s[1:]
	hasDigit := false
	for _, c := range rest {
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c < 'A' || c > 'Z' {
			return false // not an uppercase letter
		}
	}
	return hasDigit
}

// resolveUserHandle resolves a Slack @handle to a user ID. The leading "@" may
// be included or omitted. It matches the username (the @name, which is unique)
// first, then falls back to display name, and errors if the handle is unknown
// or ambiguous.
func (c *Client) resolveUserHandle(handle string) (string, error) {
	handle = strings.TrimPrefix(handle, "@")
	if handle == "" {
		return "", fmt.Errorf("user handle cannot be empty")
	}

	users, err := c.ListAllUsers()
	if err != nil {
		return "", fmt.Errorf("failed to list users: %w", err)
	}

	var byName, byDisplay []User
	for _, u := range users {
		switch {
		case strings.EqualFold(u.Name, handle):
			byName = append(byName, u)
		case strings.EqualFold(u.Profile.DisplayName, handle):
			byDisplay = append(byDisplay, u)
		}
	}

	matches := byName
	if len(matches) == 0 {
		matches = byDisplay
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("user '@%s' not found. Use 'slck users search' to find a user", handle)
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, len(matches))
		for i, u := range matches {
			ids[i] = u.ID
		}
		return "", fmt.Errorf("user '@%s' is ambiguous (matches %s); use the user ID instead", handle, strings.Join(ids, ", "))
	}
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
