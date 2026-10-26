package messages

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/render"
)

// NewCmd creates the messages command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "messages",
		Aliases: []string{"msg", "m"},
		Short:   "Manage Slack messages",
	}

	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newHistoryCmd())
	cmd.AddCommand(newThreadCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newReactCmd())
	cmd.AddCommand(newUnreactCmd())
	cmd.AddCommand(newPermalinkCmd())

	return cmd
}

// Rendering is shared with other commands through internal/render.
var (
	formatTimestamp    = render.Timestamp
	flatten            = render.Flatten
	messageBody        = render.MessageBody
	messageAuthor      = render.MessageAuthor
	indentContinuation = render.IndentContinuation
	renderFiles        = render.Files
	renderReactions    = render.Reactions
)

// truncate shortens a string to maxLen, replacing newlines with spaces
func truncate(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// unescapeShellChars removes backslash escaping from common shell-escaped characters.
// Some shells (particularly zsh) escape certain characters like ! even within single quotes.
// This function restores the intended text by removing these unnecessary escapes.
func unescapeShellChars(s string) string {
	// Unescape common shell-escaped characters
	// The most common issue is \! from zsh's bang history escaping
	s = strings.ReplaceAll(s, `\!`, `!`)
	return s
}

// maxSectionTextLen is Slack's character limit for section block text fields.
const maxSectionTextLen = 3000

// maxMessageTextLen is Slack's hard limit for the text field of a message.
// Messages exceeding this limit are silently truncated by Slack.
// See: https://docs.slack.dev/changelog/2018-truncating-really-long-messages/
const maxMessageTextLen = 40000

// validateMessageLength checks whether text exceeds Slack's message length limit.
// The forUpdate parameter controls which alternatives are suggested in the error message.
func validateMessageLength(text string, forUpdate bool) error {
	charCount := utf8.RuneCountInString(text)
	if charCount <= maxMessageTextLen {
		return nil
	}
	if forUpdate {
		return fmt.Errorf(
			"message text is %d characters, which exceeds Slack's %d character limit\n"+
				"The message would be silently truncated by Slack\n\n"+
				"Alternative:\n"+
				"  slck canvas create  Create a Slack canvas instead\n\n"+
				"To split into multiple messages, write the content to a file and chunk it yourself",
			charCount, maxMessageTextLen,
		)
	}
	return fmt.Errorf(
		"message text is %d characters, which exceeds Slack's %d character limit\n"+
			"The message would be silently truncated by Slack\n\n"+
			"Alternatives:\n"+
			"  --file <path>       Upload as a file attachment (no length limit)\n"+
			"  slck canvas create  Create a Slack canvas instead\n\n"+
			"To split into multiple messages, write the content to a file and chunk it yourself",
		charCount, maxMessageTextLen,
	)
}

// buildDefaultBlocks creates Block Kit section blocks with mrkdwn formatting.
// This provides a more refined appearance compared to plain text messages.
// Text exceeding Slack's 3000-char section limit is split across multiple blocks,
// breaking at the last newline before the limit to avoid splitting mid-line.
func buildDefaultBlocks(text string) []interface{} {
	if len(text) <= maxSectionTextLen {
		return []interface{}{
			map[string]interface{}{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": text,
				},
			},
		}
	}

	var blocks []interface{}
	remaining := text
	for len(remaining) > 0 {
		chunk := remaining
		if len(chunk) > maxSectionTextLen {
			chunk = remaining[:maxSectionTextLen]
			// Break at last newline to avoid splitting mid-line
			if idx := strings.LastIndex(chunk, "\n"); idx > 0 {
				chunk = remaining[:idx]
			}
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": chunk,
			},
		})
		remaining = remaining[len(chunk):]
		// Skip the newline we broke at
		if len(remaining) > 0 && remaining[0] == '\n' {
			remaining = remaining[1:]
		}
	}
	return blocks
}
