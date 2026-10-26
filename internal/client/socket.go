package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

// NewAppClient creates a client authenticated with the app-level token,
// which only Socket Mode methods accept.
func NewAppClient() (*Client, error) {
	st, err := keychain.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = st.Close() }()
	token, err := st.AppToken()
	if err != nil {
		return nil, err
	}
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    defaultBaseURL,
	}, nil
}

// OpenSocketConnection asks Slack for a Socket Mode WebSocket URL
// (apps.connections.open). Each URL is good for one connection.
func (c *Client) OpenSocketConnection() (string, error) {
	body, err := c.post("apps.connections.open", map[string]string{})
	if err != nil {
		return "", err
	}
	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", errors.New("apps.connections.open returned no URL")
	}
	return result.URL, nil
}
