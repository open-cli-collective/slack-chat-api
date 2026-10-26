// Package render turns Slack messages into the text slck prints: the
// "[ts] user: body" lines with their file and reaction lines.
package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// Timestamp converts a Slack timestamp to a human-readable format
func Timestamp(ts string) string {
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

// Flatten replaces newlines with spaces without truncating.
// Used by detail views (thread) where full content should be visible.
func Flatten(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

// MessageBody renders a message's readable surfaces (text, blocks,
// attachments, files) into a single body string via the shared renderer.
// preserveNewlines is true when the body came from a richer surface;
// callers that flatten plain-text in compact views should leave such
// content alone.
func MessageBody(m client.Message, resolver *client.UserResolver) (body string, preserveNewlines bool) {
	rendered := client.RenderMessage(client.MessageContent{
		Text:        m.Text,
		Blocks:      m.Blocks,
		Attachments: m.Attachments,
		Files:       m.Files,
	}, resolver)
	return rendered.Body, rendered.PreserveNewlines
}

func MessageAuthor(m client.Message, resolver *client.UserResolver) string {
	if m.User != "" {
		return resolver.Resolve(m.User)
	}
	for _, name := range []string{m.Username, m.BotProfile.Name, m.BotID, m.AppID} {
		if name != "" {
			return name
		}
	}
	return "bot"
}

// IndentContinuation replaces interior newlines with "\n\t" so that every
// line after the first is indented under the "[<ts>] <user>:" header.
// Any trailing newline is trimmed first to avoid a dangling tab.
func IndentContinuation(s string) string {
	s = strings.TrimRight(s, "\n")
	return strings.ReplaceAll(s, "\n", "\n\t")
}

// Files returns one tab-indented "[file] ..." line per attachment, each
// terminated with "\n". Returns "" when files is empty. The format gives a
// reader (human or agent) enough context to invoke `slck files download <id>`.
func Files(files []client.File) string {
	if len(files) == 0 {
		return ""
	}
	var b strings.Builder
	for _, f := range files {
		name := f.Title
		if name == "" {
			name = f.Name
		}
		if name == "" {
			// Slack can return both title and name empty for anonymous
			// snippets or certain file-sharing events. Fall back to the
			// file ID so the line never renders as "[file]  (...)".
			name = f.ID
		}
		b.WriteString("\t[file] ")
		b.WriteString(name)
		b.WriteString(" (")
		// Slack occasionally omits filetype; drop the type clause rather
		// than emit "(, 411 B)".
		if f.Filetype != "" {
			b.WriteString(f.Filetype)
			b.WriteString(", ")
		}
		b.WriteString(output.HumanSize(f.Size))
		b.WriteString(") — slck files download ")
		b.WriteString(f.ID)
		b.WriteString("\n")
	}
	return b.String()
}

// Reactions returns one tab-indented "reactions:" line listing each
// emoji with its count and the names of the users who reacted, terminated
// with "\n". Returns "" when there are no reactions. Slack caps the users
// array on large reactions, so any users beyond it are summarized as
// "+N more".
func Reactions(reactions []client.Reaction, resolver *client.UserResolver) string {
	if len(reactions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(reactions))
	for _, r := range reactions {
		names := make([]string, 0, len(r.Users)+1)
		for _, u := range r.Users {
			names = append(names, resolver.Resolve(u))
		}
		if extra := r.Count - len(r.Users); extra > 0 && len(r.Users) > 0 {
			names = append(names, fmt.Sprintf("+%d more", extra))
		}
		part := fmt.Sprintf(":%s: %d", r.Name, r.Count)
		if len(names) > 0 {
			part += " (" + strings.Join(names, ", ") + ")"
		}
		parts = append(parts, part)
	}
	return "\treactions: " + strings.Join(parts, ", ") + "\n"
}

// MessageList renders "[ts] user: body" lines for each message, with
// continuation indentation for multi-line bodies, edited markers, and file
// attachment and reaction lines. Shared by `messages thread`, `messages read` and
// the watch state `slck watch` keeps.
func MessageList(messages []client.Message, resolver *client.UserResolver) string {
	var b strings.Builder
	for _, m := range messages {
		ts := Timestamp(m.TS)
		body, preserveNewlines := MessageBody(m, resolver)
		var text string
		if preserveNewlines {
			text = IndentContinuation(body)
		} else {
			text = Flatten(body)
		}
		name := MessageAuthor(m, resolver)
		edited := ""
		if m.Edited != nil {
			edited = " [edited]"
		}
		if edited != "" {
			if idx := strings.Index(text, "\n"); idx >= 0 {
				text = text[:idx] + edited + text[idx:]
				edited = ""
			}
		}
		fmt.Fprintf(&b, "[%s] %s: %s%s\n", ts, name, text, edited)
		b.WriteString(Files(m.Files))
		b.WriteString(Reactions(m.Reactions, resolver))
	}
	return b.String()
}
