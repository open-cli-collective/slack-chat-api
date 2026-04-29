// Package messageref parses and formats Slack message refs.
//
// A message ref is the API-level composite identity for a single message in
// a channel: <channel_id>/<ts>. It exists so CLI commands can hand a single
// command-ready string between search output and read commands without
// exposing the fact that lower-level Slack APIs need channel and ts as
// separate fields.
package messageref

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

// conversationIDPattern accepts the full set of Slack conversation IDs used as
// the channel portion of a message ref: public channels (C), private/group
// channels (G), and DMs (D). validate.ChannelID rejects D, which is correct
// for channel-only commands but too narrow for refs sourced from search
// (which can hit DMs with --in @user).
var conversationIDPattern = regexp.MustCompile(`^[CGD][A-Z0-9]+$`)

// Ref is a parsed message ref.
type Ref struct {
	ChannelID string // e.g. "C02DF3BEUGN"
	TS        string // canonical API ts, e.g. "1777469221.721439"
}

// String returns the canonical "<channel_id>/<ts>" form.
func (r Ref) String() string {
	return r.ChannelID + "/" + r.TS
}

// permalinkPattern matches Slack message permalinks.
// Example: https://workspace.slack.com/archives/C02DF3BEUGN/p1777469221721439
var permalinkPattern = regexp.MustCompile(`^https?://[^/]+/archives/([A-Z0-9]+)/p(\d{16})(?:\?.*)?$`)

// Parse accepts either a permalink or "<channel_id>/<ts>" and returns a Ref.
// The ts portion may be in API form (1234567890.123456) or p-prefixed form
// (p1234567890123456); both are normalized to API form on the resulting Ref.
func Parse(input string) (Ref, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Ref{}, fmt.Errorf("empty message ref")
	}

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		m := permalinkPattern.FindStringSubmatch(input)
		if m == nil {
			return Ref{}, fmt.Errorf("invalid Slack permalink %q: expected https://<workspace>/archives/<channel>/p<digits>", input)
		}
		channel := m[1]
		digits := m[2]
		ts := digits[:10] + "." + digits[10:]
		if err := validateConversationID(channel); err != nil {
			return Ref{}, err
		}
		if err := validate.Timestamp(ts); err != nil {
			return Ref{}, err
		}
		return Ref{ChannelID: channel, TS: ts}, nil
	}

	parts := strings.SplitN(input, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Ref{}, fmt.Errorf("invalid message ref %q: expected <channel_id>/<ts> or a Slack permalink", input)
	}

	channel := parts[0]
	ts := validate.NormalizeTimestamp(parts[1])

	if err := validateConversationID(channel); err != nil {
		return Ref{}, err
	}
	if err := validate.Timestamp(ts); err != nil {
		return Ref{}, err
	}

	return Ref{ChannelID: channel, TS: ts}, nil
}

func validateConversationID(id string) error {
	if !conversationIDPattern.MatchString(id) {
		return fmt.Errorf("invalid conversation ID %q: must start with C, G, or D", id)
	}
	return nil
}
