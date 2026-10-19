package apps

import (
	"errors"
	"fmt"
	"time"

	"github.com/open-cli-collective/slack-chat-api/internal/apptoken"
	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

// tokenStore loads and saves the app configuration token pair's stored JSON.
type tokenStore interface {
	Load() (string, error)
	Save(value string) error
}

// session runs app manifest calls with a current access token, rotating the
// pair before a call when it is near expiry and once more when Slack rejects
// the access token. Every rotation is saved before the token is used, since
// Slack may invalidate the previous refresh token.
type session struct {
	api   *client.AppsClient
	store tokenStore
	now   func() time.Time
}

func (s *session) do(call func(token string) error) error {
	raw, err := s.store.Load()
	if err != nil {
		return err
	}
	pair, err := apptoken.Parse(raw)
	if err != nil {
		return fmt.Errorf("stored %w; store a new pair with `slck set-credential --key app_config_token --stdin`", err)
	}
	rotated := false
	if pair.NeedsRotation(s.now()) {
		if pair, err = s.rotate(pair); err != nil {
			return err
		}
		rotated = true
	}
	err = call(pair.Token)
	var apiErr *client.APIError
	if rotated || !errors.As(err, &apiErr) || !apiErr.ExpiredToken() {
		return err
	}
	if pair, err = s.rotate(pair); err != nil {
		return err
	}
	return call(pair.Token)
}

func (s *session) rotate(pair apptoken.Pair) (apptoken.Pair, error) {
	r, err := s.api.RotateToken(pair.RefreshToken)
	if err != nil {
		return pair, fmt.Errorf("rotate app configuration token: %w; generate a new pair at api.slack.com/apps if the refresh token was revoked", err)
	}
	next := apptoken.Pair{Token: r.Token, RefreshToken: r.RefreshToken, Exp: r.Exp, TeamID: r.TeamID, UserID: r.UserID}
	value, err := next.Encode()
	if err != nil {
		return pair, err
	}
	if err := s.store.Save(value); err != nil {
		return pair, fmt.Errorf("save rotated app configuration token: %w", err)
	}
	return next, nil
}

// keyringStore is the tokenStore backed by slck's keyring bundle.
type keyringStore struct{}

func (keyringStore) Load() (string, error) {
	st, err := keychain.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = st.Close() }()
	return st.AppConfigToken()
}

func (keyringStore) Save(value string) error {
	st, err := keychain.Open()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	return st.SetAppConfigToken(value)
}

func newSession() *session {
	return &session{api: client.NewAppsClient(), store: keyringStore{}, now: time.Now}
}
