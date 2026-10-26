package client

import (
	"regexp"
	"sync"
)

// UserResolver resolves Slack user IDs to display names with caching.
type UserResolver struct {
	client *Client
	names  *nameCache
}

type nameCache struct {
	mu sync.Mutex
	m  map[string]string
}

var mentionRegex = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)

// NewUserResolver creates a resolver backed by the given client.
func NewUserResolver(c *Client) *UserResolver {
	return &UserResolver{client: c, names: &nameCache{m: make(map[string]string)}}
}

// CacheOnly returns a resolver that shares r's cache but never calls Slack:
// an ID r has not resolved yet stays an ID.
func (r *UserResolver) CacheOnly() *UserResolver {
	if r == nil {
		return nil
	}
	return &UserResolver{names: r.names}
}

// Resolve returns a display name for the given user ID.
// It returns the ID unchanged if the lookup fails or the resolver is nil.
func (r *UserResolver) Resolve(userID string) string {
	if userID == "" {
		return userID
	}
	if r == nil || r.names == nil {
		return userID
	}

	r.names.mu.Lock()
	if name, ok := r.names.m[userID]; ok {
		r.names.mu.Unlock()
		return name
	}
	r.names.mu.Unlock()
	if r.client == nil {
		return userID
	}

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

	r.names.mu.Lock()
	r.names.m[userID] = name
	r.names.mu.Unlock()

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
