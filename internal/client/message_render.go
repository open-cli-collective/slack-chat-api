package client

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// MentionResolver is the renderer's only I/O boundary. *UserResolver
// satisfies it; the renderer is otherwise pure given a nil or fake resolver.
type MentionResolver interface {
	Resolve(userID string) string
	ResolveMentions(text string) string
}

// MessageContent is the readable surface of a Slack message that the
// renderer walks. Search results and channel history both populate it.
type MessageContent struct {
	Text        string
	Blocks      []Block
	Attachments []Attachment
	Files       []File
}

// RenderedMessage is the renderer's output. PreserveNewlines is true when
// the body came from a richer surface (blocks/attachments/files); callers
// that flatten plain-text in compact views should leave PreserveNewlines
// content alone.
type RenderedMessage struct {
	Body             string
	PreserveNewlines bool
}

// Attachment is the legacy pre-Block-Kit message attachment shape. Only
// the readable fields are typed; the rest round-trips via raw bytes so
// --output json stays byte-identical at unmodeled keys (color, mrkdwn_in,
// callback_id, image_url, actions, ts, etc.).
type Attachment struct {
	Pretext    string            `json:"pretext,omitempty"`
	Title      string            `json:"title,omitempty"`
	Text       string            `json:"text,omitempty"`
	Fallback   string            `json:"fallback,omitempty"`
	AuthorName string            `json:"author_name,omitempty"`
	Footer     string            `json:"footer,omitempty"`
	Fields     []AttachmentField `json:"fields,omitempty"`
	Blocks     []Block           `json:"blocks,omitempty"`

	raw json.RawMessage
}

// AttachmentField is one entry in attachment.fields[].
type AttachmentField struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

// UnmarshalJSON captures the raw bytes alongside the typed fields so that
// attachments with fields outside the typed struct can be re-emitted
// without loss.
func (a *Attachment) UnmarshalJSON(data []byte) error {
	a.raw = append(a.raw[:0], data...)
	type alias Attachment
	return json.Unmarshal(data, (*alias)(a))
}

// MarshalJSON emits the original raw bytes when available so unmodeled
// fields survive an unmarshal/marshal cycle. Falls back to the typed
// alias for attachments built in code (no raw captured).
func (a Attachment) MarshalJSON() ([]byte, error) {
	if len(a.raw) > 0 {
		return a.raw, nil
	}
	type alias Attachment
	return json.Marshal(alias(a))
}

// RenderMessage produces a token-dense plain-text rendering of a Slack
// message's readable surfaces. Walks blocks, attachments, then files.
// Falls back to mention-resolved Text when no richer surface yields
// content. Top-level Text is prepended when the richer surfaces rendered
// content but Text isn't a duplicate of any individual rendered piece.
func RenderMessage(c MessageContent, resolver MentionResolver) RenderedMessage {
	pieces := make([]string, 0, 3)
	if rendered := renderBlocksWithResolver(c.Blocks, resolver); rendered != "" {
		pieces = append(pieces, rendered)
	}
	if rendered := renderAttachments(c.Attachments, resolver); rendered != "" {
		pieces = append(pieces, rendered)
	}
	if rendered := renderFilesText(c.Files); rendered != "" {
		pieces = append(pieces, rendered)
	}

	if len(pieces) == 0 {
		body := c.Text
		if resolver != nil {
			body = resolver.ResolveMentions(c.Text)
		}
		return RenderedMessage{Body: body, PreserveNewlines: false}
	}

	richer := strings.Join(pieces, "\n")
	resolvedText := c.Text
	if resolver != nil {
		resolvedText = resolver.ResolveMentions(c.Text)
	}

	if strings.TrimSpace(resolvedText) != "" && !textDuplicatesAnyPiece(resolvedText, pieces) {
		return RenderedMessage{Body: resolvedText + "\n" + richer, PreserveNewlines: true}
	}
	return RenderedMessage{Body: richer, PreserveNewlines: true}
}

// textDuplicatesAnyPiece reports whether text matches (after whitespace
// normalization) the rendering of any single richer piece. Per-piece
// rather than aggregate comparison so a multi-surface message doesn't
// double-print text that mirrors only the blocks rendering.
func textDuplicatesAnyPiece(text string, pieces []string) bool {
	norm := normalizeWhitespace(text)
	if norm == "" {
		return true
	}
	for _, p := range pieces {
		if normalizeWhitespace(p) == norm {
			return true
		}
	}
	return false
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// renderBlocksWithResolver bridges the MentionResolver interface to the
// existing *UserResolver-typed RenderBlocks. nil resolver passes through.
func renderBlocksWithResolver(blocks []Block, resolver MentionResolver) string {
	if ur, ok := resolver.(*UserResolver); ok {
		return RenderBlocks(blocks, ur)
	}
	// Either nil or a non-UserResolver implementation. RenderBlocks today
	// is parameterized on *UserResolver, but the only methods it invokes
	// are nil-safe. Pass nil through; future tests using a fake resolver
	// can call RenderBlocks directly.
	return RenderBlocks(blocks, nil)
}

func renderAttachments(atts []Attachment, resolver MentionResolver) string {
	if len(atts) == 0 {
		return ""
	}
	var rendered []string
	for _, a := range atts {
		if r := renderAttachment(a, resolver); r != "" {
			rendered = append(rendered, r)
		}
	}
	return strings.Join(rendered, "\n")
}

func renderAttachment(a Attachment, resolver MentionResolver) string {
	var pieces []string
	addNonEmpty := func(s string) {
		if s != "" {
			pieces = append(pieces, s)
		}
	}

	addNonEmpty(a.Pretext)
	addNonEmpty(a.Title)
	addNonEmpty(a.Text)

	if line := renderAttachmentFields(a.Fields); line != "" {
		pieces = append(pieces, line)
	}

	if blocks := renderBlocksWithResolver(a.Blocks, resolver); blocks != "" {
		pieces = append(pieces, blocks)
	}

	addNonEmpty(a.Footer)

	if len(pieces) == 0 {
		// Fallback only when nothing else from the attachment rendered.
		return strings.TrimSpace(a.Fallback)
	}
	return strings.Join(pieces, "\n")
}

func renderAttachmentFields(fields []AttachmentField) string {
	if len(fields) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fields {
		title := strings.TrimSpace(f.Title)
		value := strings.TrimSpace(f.Value)
		switch {
		case title != "" && value != "":
			parts = append(parts, title+": "+value)
		case value != "":
			parts = append(parts, value)
		case title != "":
			parts = append(parts, title)
		}
	}
	return strings.Join(parts, " · ")
}

// plainTextSnippetCap caps the rendered length of file.plain_text bodies
// so search tables don't blow up when a snippet contains a large file.
const plainTextSnippetCap = 1024

func renderFilesText(files []File) string {
	if len(files) == 0 {
		return ""
	}
	var rendered []string
	for _, f := range files {
		if r := renderFileText(f); r != "" {
			rendered = append(rendered, r)
		}
	}
	return strings.Join(rendered, "\n")
}

func renderFileText(f File) string {
	var pieces []string
	if f.InitialComment != nil {
		if c := strings.TrimSpace(f.InitialComment.Comment); c != "" {
			pieces = append(pieces, c)
		}
	}
	if pt := strings.TrimSpace(f.PlainText); pt != "" {
		pieces = append(pieces, capRunes(pt, plainTextSnippetCap))
	}

	if len(pieces) > 0 {
		return strings.Join(pieces, "\n")
	}

	// Fallback label: title → name → id. A file with no textual content
	// still appears as "file: <name>" so the message isn't silently
	// invisible.
	for _, label := range []string{f.Title, f.Name, f.ID} {
		if label = strings.TrimSpace(label); label != "" {
			return "file: " + label
		}
	}
	return ""
}

// capRunes truncates s to at most n runes, appending an ellipsis marker
// when truncation occurred. Rune-safe so multi-byte characters aren't
// split mid-sequence.
func capRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}
