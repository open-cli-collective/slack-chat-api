// Package apptoken models a Slack app configuration token pair: the
// short-lived access token the apps.manifest.* methods accept, the refresh
// token that tooling.tokens.rotate exchanges for a new pair, and the access
// token's expiry.
//
// The pair is stored in the keyring as one JSON value under
// keychain.KeyAppConfigToken, so a rotation replaces both tokens in a single
// write. The stored shape is a superset of the ingress contract.
package apptoken

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// RotateWithin is how close to expiry an access token is rotated before use.
// Slack issues access tokens for 12 hours.
const RotateWithin = 5 * time.Minute

// Pair is a stored app configuration token pair. Token and RefreshToken are
// secrets and are never printed.
type Pair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	// Exp is the access token's expiry in Unix seconds.
	Exp int64 `json:"exp"`
	// TeamID and UserID are recorded from the first rotation; the ingress
	// contract does not carry them.
	TeamID string `json:"team_id,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// Parse validates and decodes an ingress value or a stored value. The
// ingress contract is exactly {"token", "refresh_token", "exp"}, the field
// names tooling.tokens.rotate answers with. Errors never include a token.
func Parse(raw string) (Pair, error) {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	var p Pair
	if err := dec.Decode(&p); err != nil {
		return Pair{}, errors.New(`app configuration token must be a JSON object {"token": "...", "refresh_token": "...", "exp": <unix seconds>}`)
	}
	if !strings.HasPrefix(p.Token, "xoxe.") {
		return Pair{}, errors.New(`app configuration token: "token" is not an app configuration access token (xoxe.*)`)
	}
	if !strings.HasPrefix(p.RefreshToken, "xoxe-") {
		return Pair{}, errors.New(`app configuration token: "refresh_token" is not an app configuration refresh token (xoxe-*)`)
	}
	if p.Exp <= 0 {
		return Pair{}, errors.New(`app configuration token: "exp" must be the access token's expiry in Unix seconds`)
	}
	return p, nil
}

// Encode returns the stored JSON value.
func (p Pair) Encode() (string, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("encode app configuration token: %w", err)
	}
	return string(b), nil
}

// Expiry returns the access token's expiry.
func (p Pair) Expiry() time.Time { return time.Unix(p.Exp, 0) }

// NeedsRotation reports whether the access token expires within RotateWithin
// of now.
func (p Pair) NeedsRotation(now time.Time) bool {
	return !now.Add(RotateWithin).Before(p.Expiry())
}
