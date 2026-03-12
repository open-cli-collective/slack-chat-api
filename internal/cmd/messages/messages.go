package messages

import (
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
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
	cmd.AddCommand(newReactCmd())
	cmd.AddCommand(newUnreactCmd())

	return cmd
}

// formatTimestamp converts a Slack timestamp to a human-readable format
func formatTimestamp(ts string) string {
	// Slack timestamps are Unix timestamps with decimals
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return ts
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ts
	}
	t := time.Unix(sec, 0)
	return t.Format("2006-01-02 15:04")
}

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

// buildDefaultBlocks creates a Block Kit section block with mrkdwn formatting.
// This provides a more refined appearance compared to plain text messages.
func buildDefaultBlocks(text string) []interface{} {
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
