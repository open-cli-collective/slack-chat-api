package validate

import (
	"fmt"
	"regexp"
	"strings"
)

// slackURLTimestampRegex matches Slack message URLs and captures the p-prefixed timestamp
// Example: https://workspace.slack.com/archives/C123/p1234567890123456
var slackURLTimestampRegex = regexp.MustCompile(`/p(\d{16})(?:\?|$)`)

var (
	channelIDRegex = regexp.MustCompile(`^[CG][A-Z0-9]+$`)
	userIDRegex    = regexp.MustCompile(`^[UW][A-Z0-9]+$`)
	timestampRegex = regexp.MustCompile(`^\d+\.\d+$`)
)

// ChannelID validates that the given string is a valid Slack channel ID.
// Channel IDs start with C (public) or G (private/group).
func ChannelID(id string) error {
	if !channelIDRegex.MatchString(id) {
		return fmt.Errorf("invalid channel ID %q: must start with C or G (e.g., C01234ABCDE)", id)
	}
	return nil
}

// UserID validates that the given string is a valid Slack user ID.
// User IDs start with U (regular user) or W (enterprise user).
func UserID(id string) error {
	if !userIDRegex.MatchString(id) {
		return fmt.Errorf("invalid user ID %q: must start with U or W (e.g., U01234ABCDE)", id)
	}
	return nil
}

// NormalizeTimestamp converts various Slack timestamp formats to the standard API format.
// Accepts:
//   - Standard API format: "1234567890.123456" (returned as-is)
//   - P-prefixed format: "p1234567890123456" (from Slack URLs)
//   - Full Slack URL: "https://workspace.slack.com/archives/C123/p1234567890123456"
//
// Returns the normalized timestamp in API format, or the original input if no conversion applies.
func NormalizeTimestamp(input string) string {
	input = strings.TrimSpace(input)

	// Check for full Slack URL containing /p<timestamp>
	if strings.Contains(input, "/p") {
		if matches := slackURLTimestampRegex.FindStringSubmatch(input); len(matches) == 2 {
			digits := matches[1] // 16 digits without 'p'
			return digits[:10] + "." + digits[10:]
		}
	}

	// Check for p-prefixed format (p + 16 digits)
	if strings.HasPrefix(input, "p") && len(input) == 17 {
		digits := input[1:] // Remove 'p'
		// Verify all characters are digits
		allDigits := true
		for _, c := range digits {
			if c < '0' || c > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return digits[:10] + "." + digits[10:]
		}
	}

	// Return as-is (already in API format or invalid - validation will catch it)
	return input
}

// Timestamp validates that the given string is a valid Slack message timestamp.
// Timestamps are in the format "1234567890.123456".
// Also accepts p-prefixed format and full Slack URLs, which are normalized first.
func Timestamp(ts string) error {
	normalized := NormalizeTimestamp(ts)
	if !timestampRegex.MatchString(normalized) {
		return fmt.Errorf("invalid timestamp %q: must be format 1234567890.123456, p1234567890123456, or a Slack message URL", ts)
	}
	return nil
}

// Emoji normalizes an emoji name by stripping surrounding colons.
// Returns the cleaned emoji name.
func Emoji(emoji string) string {
	return strings.Trim(emoji, ":")
}

// Limit validates that the given limit is within acceptable bounds.
func Limit(limit int) error {
	if limit < 1 {
		return fmt.Errorf("invalid limit %d: must be at least 1", limit)
	}
	if limit > 1000 {
		return fmt.Errorf("invalid limit %d: must be at most 1000", limit)
	}
	return nil
}
