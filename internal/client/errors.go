package client

import (
	"fmt"
	"strings"
)

// errorHints maps Slack API error codes to helpful hints.
var errorHints = map[string]string{
	"channel_not_found":    "Verify the channel ID is correct. Use 'slack-chat-api channels list' to find channel IDs.",
	"not_in_channel":       "The bot must be invited to the channel. Use /invite @yourbot in Slack.",
	"invalid_auth":         "Token is invalid or expired. Run 'slack-chat-api config set-token' to set a new token.",
	"token_revoked":        "Token has been revoked. Run 'slack-chat-api config set-token' to set a new token.",
	"ratelimited":          "Rate limit exceeded. Wait a moment and try again.",
	"user_not_found":       "Verify the user ID is correct. Use 'slack-chat-api users list' to find user IDs.",
	"message_not_found":    "Message not found. Verify the channel ID and timestamp are correct.",
	"cant_delete_message":  "Cannot delete this message. You can only delete messages sent by the bot.",
	"cant_update_message":  "Cannot update this message. You can only update messages sent by the bot.",
	"already_archived":     "Channel is already archived.",
	"not_archived":         "Channel is not archived.",
	"name_taken":           "A channel with this name already exists.",
	"invalid_name":         "Invalid channel name. Use lowercase letters, numbers, and hyphens only.",
	"no_permission":        "The bot lacks permission for this action. Check the app's OAuth scopes.",
	"missing_scope":        "Missing required OAuth scope. Update your app's permissions at api.slack.com/apps.",
	"account_inactive":     "The user account is inactive or disabled.",
	"is_archived":          "Cannot perform this action on an archived channel.",
	"too_many_attachments": "Message has too many attachments. Reduce and try again.",
	"msg_too_long":         "Message is too long. Maximum is 40,000 characters.",
}

// WrapError wraps a Slack API error with context and a helpful hint if available.
func WrapError(operation string, err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Check for known error codes
	for code, hint := range errorHints {
		if strings.Contains(errStr, code) {
			return fmt.Errorf("%s: %w\nHint: %s", operation, err, hint)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}
