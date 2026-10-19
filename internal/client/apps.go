package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AppsClient calls the app manifest methods (apps.manifest.*) and
// tooling.tokens.rotate. Unlike Client it carries no token of its own: the
// manifest methods take an app configuration access token, which rotates,
// so each call is given the current one.
type AppsClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewAppsClient returns an AppsClient for the Slack API.
func NewAppsClient() *AppsClient {
	return NewAppsClientWithConfig(defaultBaseURL, nil)
}

// NewAppsClientWithConfig returns an AppsClient for baseURL; tests pass an
// httptest server.
func NewAppsClientWithConfig(baseURL string, httpClient *http.Client) *AppsClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &AppsClient{httpClient: httpClient, baseURL: baseURL}
}

// ManifestError is one problem apps.manifest.validate, create or update
// found in a manifest. Pointer is a JSON pointer into the manifest.
type ManifestError struct {
	Message string `json:"message"`
	Pointer string `json:"pointer"`
}

// AppsAPIError is a Slack answer with ok=false from an AppsClient method.
// Errors carries the manifest problems Slack lists for invalid_manifest.
// Client (the bot and user token client) does not return it; its errors are
// matched with IsSlackError.
type AppsAPIError struct {
	Method string
	Code   string
	Errors []ManifestError
}

func (e *AppsAPIError) Error() string {
	msg := fmt.Sprintf("slack API error: %s (%s)", e.Code, e.Method)
	for _, m := range e.Errors {
		msg += fmt.Sprintf("\n  %s: %s", m.Pointer, m.Message)
	}
	return msg
}

// ExpiredToken reports whether the error means the access token no longer
// works, so a rotation may recover it.
func (e *AppsAPIError) ExpiredToken() bool {
	return e.Code == "token_expired" || e.Code == "invalid_auth"
}

// AppCredentials are the secrets apps.manifest.create returns for a new
// app. They are written only to a file the caller names, never printed.
type AppCredentials struct {
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	VerificationToken string `json:"verification_token"`
	SigningSecret     string `json:"signing_secret"`
}

// CreatedApp is apps.manifest.create's answer.
type CreatedApp struct {
	AppID             string         `json:"app_id"`
	Credentials       AppCredentials `json:"credentials"`
	OAuthAuthorizeURL string         `json:"oauth_authorize_url"`
}

// RotatedToken is tooling.tokens.rotate's answer.
type RotatedToken struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	TeamID       string `json:"team_id"`
	UserID       string `json:"user_id"`
	Exp          int64  `json:"exp"`
}

// ExportManifest returns an app's manifest exactly as Slack sends it.
func (c *AppsClient) ExportManifest(token, appID string) (json.RawMessage, error) {
	var resp struct {
		Manifest json.RawMessage `json:"manifest"`
	}
	if err := c.call("apps.manifest.export", token, url.Values{"app_id": {appID}}, &resp); err != nil {
		return nil, err
	}
	return resp.Manifest, nil
}

// ValidateManifest checks a manifest, against an existing app when appID is
// set. A manifest with problems returns an *AppsAPIError listing them.
func (c *AppsClient) ValidateManifest(token string, manifest json.RawMessage, appID string) error {
	params := url.Values{"manifest": {string(manifest)}}
	if appID != "" {
		params.Set("app_id", appID)
	}
	return c.call("apps.manifest.validate", token, params, nil)
}

// CreateApp creates an app from a manifest.
func (c *AppsClient) CreateApp(token string, manifest json.RawMessage) (CreatedApp, error) {
	var resp CreatedApp
	err := c.call("apps.manifest.create", token, url.Values{"manifest": {string(manifest)}}, &resp)
	return resp, err
}

// UpdateApp replaces an app's manifest and reports whether its scopes
// changed, in which case the app must be reinstalled for them to apply.
func (c *AppsClient) UpdateApp(token, appID string, manifest json.RawMessage) (bool, error) {
	var resp struct {
		PermissionsUpdated bool `json:"permissions_updated"`
	}
	err := c.call("apps.manifest.update", token, url.Values{"app_id": {appID}, "manifest": {string(manifest)}}, &resp)
	return resp.PermissionsUpdated, err
}

// DeleteApp permanently deletes an app.
func (c *AppsClient) DeleteApp(token, appID string) error {
	return c.call("apps.manifest.delete", token, url.Values{"app_id": {appID}}, nil)
}

// RotateToken exchanges a refresh token for a new configuration token pair.
// It takes no access token.
func (c *AppsClient) RotateToken(refreshToken string) (RotatedToken, error) {
	var resp RotatedToken
	err := c.call("tooling.tokens.rotate", "", url.Values{"refresh_token": {refreshToken}}, &resp)
	return resp, err
}

// call POSTs form-encoded params, which every method here accepts, and
// decodes the answer into out when it is ok.
func (c *AppsClient) call(method, token string, params url.Values, out any) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", c.baseURL, method), strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("%s: rate limited by Slack; retry after %s seconds", method, resp.Header.Get("Retry-After"))
	}
	var status struct {
		OK     bool            `json:"ok"`
		Error  string          `json:"error"`
		Errors []ManifestError `json:"errors"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return fmt.Errorf("%s: unexpected response (HTTP %d)", method, resp.StatusCode)
	}
	if !status.OK {
		return &AppsAPIError{Method: method, Code: status.Error, Errors: status.Errors}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: decode response: %w", method, err)
	}
	return nil
}
