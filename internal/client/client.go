package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

const defaultBaseURL = "https://slack.com/api"

// Client handles Slack API interactions
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

// New creates a new Slack client
func New() (*Client, error) {
	token, err := keychain.GetAPIToken()
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    defaultBaseURL,
	}, nil
}

// NewWithConfig creates a new Slack client with custom configuration.
// This is primarily used for testing with httptest servers.
func NewWithConfig(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		httpClient: httpClient,
		token:      token,
		baseURL:    baseURL,
	}
}

// NewUserClient creates a new Slack client using the user token (for search)
func NewUserClient() (*Client, error) {
	token, err := keychain.GetUserToken()
	if err != nil {
		return nil, fmt.Errorf("user token required for search: %w", err)
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    defaultBaseURL,
	}, nil
}

// SlackResponse represents a generic Slack API response
type SlackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (c *Client) get(endpoint string, params url.Values) (result []byte, err error) {
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	if params != nil {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var slackResp SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		return nil, err
	}

	if !slackResp.OK {
		return nil, fmt.Errorf("slack API error: %s", slackResp.Error)
	}

	return body, nil
}

func (c *Client) post(endpoint string, data interface{}) (result []byte, err error) {
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, endpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var slackResp SlackResponse
	if err := json.Unmarshal(body, &slackResp); err != nil {
		return nil, err
	}

	if !slackResp.OK {
		return nil, fmt.Errorf("slack API error: %s", slackResp.Error)
	}

	return body, nil
}

// Channel represents a Slack channel
type Channel struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsPrivate  bool   `json:"is_private"`
	IsArchived bool   `json:"is_archived"`
	Topic      struct {
		Value string `json:"value"`
	} `json:"topic"`
	Purpose struct {
		Value string `json:"value"`
	} `json:"purpose"`
	NumMembers int `json:"num_members"`
}

// User represents a Slack user
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	IsAdmin  bool   `json:"is_admin"`
	IsBot    bool   `json:"is_bot"`
	Profile  struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		StatusText  string `json:"status_text"`
		StatusEmoji string `json:"status_emoji"`
	} `json:"profile"`
}

// Message represents a Slack message
type Message struct {
	Type       string `json:"type"`
	User       string `json:"user"`
	Text       string `json:"text"`
	TS         string `json:"ts"`
	ThreadTS   string `json:"thread_ts,omitempty"`
	ReplyCount int    `json:"reply_count,omitempty"`
}

// Team represents workspace info
type Team struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// SearchMatch represents a message match from search
type SearchMatch struct {
	Type    string `json:"type"`
	Channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"channel"`
	User      string `json:"user"`
	Username  string `json:"username"`
	Text      string `json:"text"`
	TS        string `json:"ts"`
	Permalink string `json:"permalink"`
}

// FileMatch represents a file match from search
type FileMatch struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Title           string `json:"title"`
	Filetype        string `json:"filetype"`
	User            string `json:"user"`
	Created         int64  `json:"created"`
	Permalink       string `json:"permalink"`
	PermalinkPublic string `json:"permalink_public,omitempty"`
}

// SearchPaging contains pagination info
type SearchPaging struct {
	Count int `json:"count"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Pages int `json:"pages"`
}

// MessageGroup contains message search results
type MessageGroup struct {
	Total   int           `json:"total"`
	Paging  SearchPaging  `json:"paging"`
	Matches []SearchMatch `json:"matches"`
}

// FileGroup contains file search results
type FileGroup struct {
	Total   int          `json:"total"`
	Paging  SearchPaging `json:"paging"`
	Matches []FileMatch  `json:"matches"`
}

// SearchResult contains search results
type SearchResult struct {
	Query    string        `json:"query"`
	Messages *MessageGroup `json:"messages,omitempty"`
	Files    *FileGroup    `json:"files,omitempty"`
}

// ListChannels returns channels up to the specified limit (handles pagination automatically)
func (c *Client) ListChannels(types string, excludeArchived bool, limit int) ([]Channel, error) {
	var allChannels []Channel
	cursor := ""
	remaining := limit

	for remaining > 0 {
		params := url.Values{}
		params.Set("exclude_archived", fmt.Sprintf("%t", excludeArchived))
		// Request up to 200 at a time (Slack recommended max), or remaining if smaller
		batchSize := remaining
		if batchSize > 200 {
			batchSize = 200
		}
		params.Set("limit", fmt.Sprintf("%d", batchSize))
		if types != "" {
			params.Set("types", types)
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.get("conversations.list", params)
		if err != nil {
			return nil, err
		}

		var result struct {
			Channels         []Channel `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allChannels = append(allChannels, result.Channels...)
		remaining -= len(result.Channels)

		if result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	// Trim to exact limit if we got more
	if len(allChannels) > limit {
		allChannels = allChannels[:limit]
	}

	return allChannels, nil
}

// GetChannelInfo returns channel details
func (c *Client) GetChannelInfo(channelID string) (*Channel, error) {
	params := url.Values{}
	params.Set("channel", channelID)

	body, err := c.get("conversations.info", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		Channel Channel `json:"channel"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.Channel, nil
}

// ListUsers returns users up to the specified limit (handles pagination automatically)
func (c *Client) ListUsers(limit int) ([]User, error) {
	var allUsers []User
	cursor := ""
	remaining := limit

	for remaining > 0 {
		params := url.Values{}
		// Request up to 200 at a time (Slack recommended max), or remaining if smaller
		batchSize := remaining
		if batchSize > 200 {
			batchSize = 200
		}
		params.Set("limit", fmt.Sprintf("%d", batchSize))
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.get("users.list", params)
		if err != nil {
			return nil, err
		}

		var result struct {
			Members          []User `json:"members"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allUsers = append(allUsers, result.Members...)
		remaining -= len(result.Members)

		if result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	// Trim to exact limit if we got more
	if len(allUsers) > limit {
		allUsers = allUsers[:limit]
	}

	return allUsers, nil
}

// GetUserInfo returns user details
func (c *Client) GetUserInfo(userID string) (*User, error) {
	params := url.Values{}
	params.Set("user", userID)

	body, err := c.get("users.info", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.User, nil
}

// SendMessage sends a message to a channel.
// Text can be empty if blocks are provided (Slack API allows this).
// The unfurl parameter controls whether link previews are shown (unfurl_links and unfurl_media).
func (c *Client) SendMessage(channel, text, threadTS string, blocks []interface{}, unfurl bool) (*Message, error) {
	data := map[string]interface{}{
		"channel":      channel,
		"unfurl_links": unfurl,
		"unfurl_media": unfurl,
	}
	// Only include text if non-empty (Slack allows omitting text when blocks are provided)
	if text != "" {
		data["text"] = text
	}
	if threadTS != "" {
		data["thread_ts"] = threadTS
	}
	if len(blocks) > 0 {
		data["blocks"] = blocks
	}

	body, err := c.post("chat.postMessage", data)
	if err != nil {
		return nil, err
	}

	var result struct {
		Message Message `json:"message"`
		TS      string  `json:"ts"`
		Channel string  `json:"channel"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	result.Message.TS = result.TS
	return &result.Message, nil
}

// UpdateMessage updates an existing message.
// The unfurl parameter controls whether link previews are shown (unfurl_links and unfurl_media).
func (c *Client) UpdateMessage(channel, ts, text string, blocks []interface{}, unfurl bool) error {
	data := map[string]interface{}{
		"channel":      channel,
		"ts":           ts,
		"text":         text,
		"unfurl_links": unfurl,
		"unfurl_media": unfurl,
	}
	if len(blocks) > 0 {
		data["blocks"] = blocks
	}

	_, err := c.post("chat.update", data)
	return err
}

// DeleteMessage deletes a message
func (c *Client) DeleteMessage(channel, ts string) error {
	data := map[string]interface{}{
		"channel": channel,
		"ts":      ts,
	}

	_, err := c.post("chat.delete", data)
	return err
}

// GetChannelHistory returns message history (handles pagination to reach requested limit)
func (c *Client) GetChannelHistory(channel string, limit int, oldest, latest string) ([]Message, error) {
	var allMessages []Message
	cursor := ""
	remaining := limit

	for remaining > 0 {
		params := url.Values{}
		params.Set("channel", channel)
		// Request up to 200 at a time (Slack recommended max)
		batchSize := remaining
		if batchSize > 200 {
			batchSize = 200
		}
		params.Set("limit", fmt.Sprintf("%d", batchSize))
		if oldest != "" {
			params.Set("oldest", oldest)
		}
		if latest != "" {
			params.Set("latest", latest)
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.get("conversations.history", params)
		if err != nil {
			return nil, err
		}

		var result struct {
			Messages         []Message `json:"messages"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allMessages = append(allMessages, result.Messages...)
		remaining -= len(result.Messages)

		if result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	// Trim to exact limit if we got more
	if len(allMessages) > limit {
		allMessages = allMessages[:limit]
	}

	return allMessages, nil
}

// GetThreadReplies returns replies to a thread (handles pagination to reach requested limit)
func (c *Client) GetThreadReplies(channel, threadTS string, limit int) ([]Message, error) {
	var allMessages []Message
	cursor := ""
	remaining := limit

	for remaining > 0 {
		params := url.Values{}
		params.Set("channel", channel)
		params.Set("ts", threadTS)
		// Request up to 200 at a time (Slack recommended max)
		batchSize := remaining
		if batchSize > 200 {
			batchSize = 200
		}
		params.Set("limit", fmt.Sprintf("%d", batchSize))
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		body, err := c.get("conversations.replies", params)
		if err != nil {
			return nil, err
		}

		var result struct {
			Messages         []Message `json:"messages"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		allMessages = append(allMessages, result.Messages...)
		remaining -= len(result.Messages)

		if result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	// Trim to exact limit if we got more
	if len(allMessages) > limit {
		allMessages = allMessages[:limit]
	}

	return allMessages, nil
}

// AddReaction adds an emoji reaction
func (c *Client) AddReaction(channel, timestamp, name string) error {
	data := map[string]interface{}{
		"channel":   channel,
		"timestamp": timestamp,
		"name":      name,
	}

	_, err := c.post("reactions.add", data)
	return err
}

// RemoveReaction removes an emoji reaction
func (c *Client) RemoveReaction(channel, timestamp, name string) error {
	data := map[string]interface{}{
		"channel":   channel,
		"timestamp": timestamp,
		"name":      name,
	}

	_, err := c.post("reactions.remove", data)
	return err
}

// GetTeamInfo returns workspace info
func (c *Client) GetTeamInfo() (*Team, error) {
	body, err := c.get("team.info", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Team Team `json:"team"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.Team, nil
}

// AuthTestResponse represents the response from auth.test API
type AuthTestResponse struct {
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	BotID  string `json:"bot_id,omitempty"`
}

// AuthTest verifies authentication and returns identity info
func (c *Client) AuthTest() (*AuthTestResponse, error) {
	body, err := c.post("auth.test", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result AuthTestResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateChannel creates a new channel
func (c *Client) CreateChannel(name string, isPrivate bool) (*Channel, error) {
	data := map[string]interface{}{
		"name":       name,
		"is_private": isPrivate,
	}

	body, err := c.post("conversations.create", data)
	if err != nil {
		return nil, err
	}

	var result struct {
		Channel Channel `json:"channel"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.Channel, nil
}

// ArchiveChannel archives a channel
func (c *Client) ArchiveChannel(channel string) error {
	data := map[string]interface{}{
		"channel": channel,
	}
	_, err := c.post("conversations.archive", data)
	return err
}

// UnarchiveChannel unarchives a channel
func (c *Client) UnarchiveChannel(channel string) error {
	data := map[string]interface{}{
		"channel": channel,
	}
	_, err := c.post("conversations.unarchive", data)
	return err
}

// SetChannelTopic sets the channel topic
func (c *Client) SetChannelTopic(channel, topic string) error {
	data := map[string]interface{}{
		"channel": channel,
		"topic":   topic,
	}
	_, err := c.post("conversations.setTopic", data)
	return err
}

// SetChannelPurpose sets the channel purpose
func (c *Client) SetChannelPurpose(channel, purpose string) error {
	data := map[string]interface{}{
		"channel": channel,
		"purpose": purpose,
	}
	_, err := c.post("conversations.setPurpose", data)
	return err
}

// InviteToChannel invites users to a channel
func (c *Client) InviteToChannel(channel string, users []string) error {
	usersStr := ""
	for i, u := range users {
		if i > 0 {
			usersStr += ","
		}
		usersStr += u
	}

	data := map[string]interface{}{
		"channel": channel,
		"users":   usersStr,
	}
	_, err := c.post("conversations.invite", data)
	return err
}

// --- Search Methods (require user token) ---

// SearchMessages searches for messages matching a query
func (c *Client) SearchMessages(query string, count, page int, sort, sortDir string, highlight, includeBots bool) (*SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("sort", sort)
	params.Set("sort_dir", sortDir)
	if highlight {
		params.Set("highlight", "true")
	}
	if includeBots {
		params.Set("search_exclude_bots", "false")
	}

	body, err := c.get("search.messages", params)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	result.Query = query
	return &result, nil
}

// SearchFiles searches for files matching a query
func (c *Client) SearchFiles(query string, count, page int, sort, sortDir string, highlight, includeBots bool) (*SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("sort", sort)
	params.Set("sort_dir", sortDir)
	if highlight {
		params.Set("highlight", "true")
	}
	if includeBots {
		params.Set("search_exclude_bots", "false")
	}

	body, err := c.get("search.files", params)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	result.Query = query
	return &result, nil
}

// SearchAll searches for both messages and files matching a query
func (c *Client) SearchAll(query string, count, page int, sort, sortDir string, highlight, includeBots bool) (*SearchResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("count", fmt.Sprintf("%d", count))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("sort", sort)
	params.Set("sort_dir", sortDir)
	if highlight {
		params.Set("highlight", "true")
	}
	if includeBots {
		params.Set("search_exclude_bots", "false")
	}

	body, err := c.get("search.all", params)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	result.Query = query
	return &result, nil
}
