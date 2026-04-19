package client

import (
	"regexp"
	"sync"
)

// UserResolver resolves Slack user IDs to display names with caching.
type UserResolver struct {
	client *Client
	cache  map[string]string
	mu     sync.Mutex
}

var mentionRegex = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)

// NewUserResolver creates a resolver backed by the given client.
func NewUserResolver(c *Client) *UserResolver {
	return &UserResolver{
		client: c,
		cache:  make(map[string]string),
	}
}

// Resolve returns a display name for the given user ID.
// It returns the ID unchanged if the lookup fails or the resolver is nil.
func (r *UserResolver) Resolve(userID string) string {
	if userID == "" {
		return userID
	}
	if r == nil || r.client == nil {
		return userID
	}

	r.mu.Lock()
	if name, ok := r.cache[userID]; ok {
		r.mu.Unlock()
		return name
	}
	r.mu.Unlock()

	user, err := r.client.GetUserInfo(userID)
	if err != nil {
		return userID
	}

	name := user.Profile.DisplayName
	if name == "" {
		name = user.RealName
	}
	if name == "" {
		name = user.Name
	}
	if name == "" {
		name = userID
	}

	r.mu.Lock()
	r.cache[userID] = name
	r.mu.Unlock()

	return name
}

// ResolveMentions replaces <@UXXXXX> mentions in text with display names.
// Returns text unchanged if the resolver is nil.
func (r *UserResolver) ResolveMentions(text string) string {
	if r == nil {
		return text
	}
	return mentionRegex.ReplaceAllStringFunc(text, func(match string) string {
		sub := mentionRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		return "@" + r.Resolve(sub[1])
	})
}
