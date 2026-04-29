package client

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustAttachments(t *testing.T, raw string) []Attachment {
	t.Helper()
	var atts []Attachment
	require.NoError(t, json.Unmarshal([]byte(raw), &atts))
	return atts
}

func TestRenderBlocks_Section(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "section",
		"text": {"type": "mrkdwn", "text": "Served Rian in #general · 20.7s · claude-sonnet-4-6"}
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Equal(t, "Served Rian in #general · 20.7s · claude-sonnet-4-6", got)
}

func TestRenderBlocks_SectionFields(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "section",
		"text": {"type": "mrkdwn", "text": "Status report"},
		"fields": [
			{"type": "mrkdwn", "text": "Cost: $0.07"},
			{"type": "mrkdwn", "text": "Tokens: 73,746"}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Equal(t, "Status report\nCost: $0.07 · Tokens: 73,746", got)
}

func TestRenderBlocks_Header(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "header",
		"text": {"type": "plain_text", "text": "Deploy Summary"}
	}]`)
	assert.Equal(t, "Deploy Summary", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_Context(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "context",
		"elements": [
			{"type": "image", "image_url": "x", "alt_text": "logo"},
			{"type": "mrkdwn", "text": "v1.2.3"}
		]
	}]`)
	assert.Equal(t, "logo v1.2.3", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_Actions(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "actions",
		"elements": [
			{"type": "button", "text": {"type": "plain_text", "text": "Approve"}},
			{"type": "button", "text": {"type": "plain_text", "text": "Reject"}}
		]
	}]`)
	assert.Equal(t, "[Approve] [Reject]", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_ImageWithAltText(t *testing.T) {
	blocks := mustBlocks(t, `[{"type": "image", "alt_text": "graph of latency", "image_url": "x"}]`)
	assert.Equal(t, "graph of latency", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_ImageWithoutAltOrTitle(t *testing.T) {
	blocks := mustBlocks(t, `[{"type": "image", "image_url": "x"}]`)
	assert.Equal(t, "", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_VideoTitleAndDescription(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "video",
		"title": {"type": "plain_text", "text": "Demo"},
		"description": {"type": "plain_text", "text": "release walkthrough"}
	}]`)
	assert.Equal(t, "Demo\nrelease walkthrough", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_DividerDropped(t *testing.T) {
	blocks := mustBlocks(t, `[
		{"type": "divider"},
		{"type": "section", "text": {"type": "mrkdwn", "text": "after"}}
	]`)
	assert.Equal(t, "after", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_BugRepro(t *testing.T) {
	// Original bug shape from #143: a single section block carries the
	// status line and m.Text is empty.
	blocks := mustBlocks(t, `[{
		"type": "section",
		"text": {"type": "mrkdwn", "text": "Served Rian in #general · 20.7s · claude-sonnet-4-6 · 2 in (73,746 cached) · 77 out · $0.07"}
	}]`)
	got := RenderMessage(MessageContent{Blocks: blocks}, nil)
	assert.Equal(t, "Served Rian in #general · 20.7s · claude-sonnet-4-6 · 2 in (73,746 cached) · 77 out · $0.07", got.Body)
	assert.True(t, got.PreserveNewlines)
}

func TestRenderMessage_FallsBackToText(t *testing.T) {
	got := RenderMessage(MessageContent{Text: "hi <@U123>"}, nil)
	assert.Equal(t, "hi <@U123>", got.Body)
	assert.False(t, got.PreserveNewlines)
}

func TestRenderMessage_EmptyEverywhere(t *testing.T) {
	got := RenderMessage(MessageContent{}, nil)
	assert.Equal(t, "", got.Body)
	assert.False(t, got.PreserveNewlines)
}

func TestRenderMessage_MultiSurfaceCombined(t *testing.T) {
	blocks := mustBlocks(t, `[{"type":"section","text":{"type":"mrkdwn","text":"block content"}}]`)
	atts := mustAttachments(t, `[{"text":"attachment content"}]`)
	files := []File{{InitialComment: &FileComment{Comment: "file commentary"}}}
	got := RenderMessage(MessageContent{
		Text:        "fallback",
		Blocks:      blocks,
		Attachments: atts,
		Files:       files,
	}, nil)
	// text != any single piece, so it gets prepended.
	assert.Equal(t, "fallback\nblock content\nattachment content\nfile commentary", got.Body)
	assert.True(t, got.PreserveNewlines)
}

func TestRenderMessage_TextDuplicatesBlocksDropped(t *testing.T) {
	blocks := mustBlocks(t, `[{"type":"section","text":{"type":"mrkdwn","text":"hello world"}}]`)
	got := RenderMessage(MessageContent{Text: "hello world", Blocks: blocks}, nil)
	assert.Equal(t, "hello world", got.Body)
	assert.True(t, got.PreserveNewlines)
}

func TestRenderMessage_TextDuplicatesOnlyOnePieceStillDropped(t *testing.T) {
	// text == blocks rendering exactly. Even though attachments add more
	// content, the text shouldn't be double-printed because it duplicates
	// one piece.
	blocks := mustBlocks(t, `[{"type":"section","text":{"type":"mrkdwn","text":"hello"}}]`)
	atts := mustAttachments(t, `[{"text":"more content"}]`)
	got := RenderMessage(MessageContent{Text: "hello", Blocks: blocks, Attachments: atts}, nil)
	assert.Equal(t, "hello\nmore content", got.Body)
	assert.True(t, got.PreserveNewlines)
}

func TestRenderMessage_TextWhitespaceNormalizedDup(t *testing.T) {
	blocks := mustBlocks(t, `[{"type":"section","text":{"type":"mrkdwn","text":"hello   world"}}]`)
	got := RenderMessage(MessageContent{Text: "hello world", Blocks: blocks}, nil)
	// Whitespace-normalized comparison treats "hello   world" == "hello world".
	assert.Equal(t, "hello   world", got.Body)
}

func TestRenderMessage_NilResolverPreservesRawMentions(t *testing.T) {
	got := RenderMessage(MessageContent{Text: "ping <@U123>"}, nil)
	assert.Equal(t, "ping <@U123>", got.Body)
}

func TestRenderMessage_NilResolverInBlocks(t *testing.T) {
	// Rich-text user element without resolver renders "@U123" (matches the
	// existing nil-safe behavior in renderInline).
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [{"type":"rich_text_section","elements":[
			{"type":"text","text":"hi "},
			{"type":"user","user_id":"U123"}
		]}]
	}]`)
	got := RenderMessage(MessageContent{Blocks: blocks}, nil)
	assert.Equal(t, "hi @U123", got.Body)
}

func TestRenderAttachments_FieldsAndNestedBlocks(t *testing.T) {
	atts := mustAttachments(t, `[{
		"pretext": "Heads up",
		"title": "Build #42",
		"text": "Failed in 12s",
		"fields": [
			{"title": "Branch", "value": "main"},
			{"title": "Commit", "value": "abc123"}
		],
		"blocks": [{"type":"section","text":{"type":"mrkdwn","text":"nested block"}}],
		"footer": "CI"
	}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, nil)
	expected := strings.Join([]string{
		"Heads up",
		"Build #42",
		"Failed in 12s",
		"Branch: main · Commit: abc123",
		"nested block",
		"CI",
	}, "\n")
	assert.Equal(t, expected, got.Body)
}

func TestRenderAttachments_AuthorNameRendered(t *testing.T) {
	atts := mustAttachments(t, `[{"author_name": "deploy-bot", "text": "shipped v1.2.3"}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, nil)
	assert.Equal(t, "deploy-bot\nshipped v1.2.3", got.Body)
}

func TestRenderAttachments_AuthorNameOnlyAttachmentDoesNotFallThroughToFallback(t *testing.T) {
	atts := mustAttachments(t, `[{"author_name": "ci-bot", "fallback": "ci-bot says..."}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, nil)
	// AuthorName renders, fallback is suppressed.
	assert.Equal(t, "ci-bot", got.Body)
}

func TestRenderMessage_MultiBlockJoinedTextIsDuplicate(t *testing.T) {
	// Slack populates m.Text as the newline-joined rendering of multiple
	// blocks for Block Kit fallback. The aggregate comparison catches
	// that case so text isn't double-printed.
	blocks := mustBlocks(t, `[
		{"type":"section","text":{"type":"mrkdwn","text":"line1"}},
		{"type":"section","text":{"type":"mrkdwn","text":"line2"}}
	]`)
	got := RenderMessage(MessageContent{Text: "line1\nline2", Blocks: blocks}, nil)
	assert.Equal(t, "line1\nline2", got.Body)
}

func TestRenderAttachments_FallbackUsedOnlyWhenEmpty(t *testing.T) {
	atts := mustAttachments(t, `[{"fallback": "plain text version"}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, nil)
	assert.Equal(t, "plain text version", got.Body)
}

func TestRenderAttachments_FallbackSuppressedWhenOtherFieldsRender(t *testing.T) {
	atts := mustAttachments(t, `[{
		"text": "real content",
		"fallback": "real content (legacy fallback)"
	}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, nil)
	// Fallback is dropped because text rendered.
	assert.Equal(t, "real content", got.Body)
}

func TestRenderFilesText_InitialCommentAndPlainText(t *testing.T) {
	files := []File{{
		InitialComment: &FileComment{Comment: "see snippet"},
		PlainText:      "package main\n\nfunc main() {}",
		Title:          "main.go",
		Name:           "main.go",
	}}
	got := RenderMessage(MessageContent{Files: files}, nil)
	assert.Equal(t, "see snippet\npackage main\n\nfunc main() {}", got.Body)
}

func TestRenderFilesText_PlainTextCapped(t *testing.T) {
	big := strings.Repeat("a", plainTextSnippetCap+500)
	files := []File{{PlainText: big}}
	got := RenderMessage(MessageContent{Files: files}, nil)
	// Capped at plainTextSnippetCap runes plus the ellipsis marker.
	assert.True(t, strings.HasSuffix(got.Body, "…"))
	assert.LessOrEqual(t, len([]rune(got.Body)), plainTextSnippetCap+1)
}

func TestRenderFilesText_LabelFallback(t *testing.T) {
	files := []File{{Title: "report.pdf", ID: "F123"}}
	got := RenderMessage(MessageContent{Files: files}, nil)
	assert.Equal(t, "file: report.pdf", got.Body)
}

func TestAttachment_JSONRoundTripPreservesUnknownFields(t *testing.T) {
	// Attachment with fields outside the typed struct (color, mrkdwn_in,
	// callback_id) must round-trip losslessly so --output json preserves
	// unknown fields the typed struct doesn't model.
	original := `{"text":"hi","color":"#36a64f","mrkdwn_in":["text"],"callback_id":"cb_42"}`
	var a Attachment
	require.NoError(t, json.Unmarshal([]byte(original), &a))

	out, err := json.Marshal(a)
	require.NoError(t, err)
	assert.JSONEq(t, original, string(out))
	assert.Contains(t, string(out), `"color":"#36a64f"`)
	assert.Contains(t, string(out), `"callback_id":"cb_42"`)
}

// fakeResolver is a non-*UserResolver implementation of MentionResolver
// used to confirm the renderer's interface contract.
type fakeResolver struct {
	names map[string]string
}

func (f *fakeResolver) Resolve(userID string) string {
	if name, ok := f.names[userID]; ok {
		return name
	}
	return userID
}

func (f *fakeResolver) ResolveMentions(text string) string {
	for id, name := range f.names {
		text = strings.ReplaceAll(text, "<@"+id+">", "@"+name)
	}
	return text
}

// TestRenderMessage_NonUserResolverHonoredForPlainText confirms a
// MentionResolver implementation that isn't *UserResolver still gets
// invoked for plain-text surfaces (section/attachment/file). Rich-text
// inline rendering is a known gap (deferred) — see the comment on
// renderBlocksWithResolver.
func TestRenderMessage_NonUserResolverHonoredForPlainText(t *testing.T) {
	resolver := &fakeResolver{names: map[string]string{"U999": "alice"}}

	blocks := mustBlocks(t, `[{
		"type": "section",
		"text": {"type": "mrkdwn", "text": "hi <@U999>"}
	}]`)
	got := RenderMessage(MessageContent{Blocks: blocks}, resolver)
	assert.Equal(t, "hi @alice", got.Body)
}

func TestRenderMessage_NonUserResolverOnAttachmentText(t *testing.T) {
	resolver := &fakeResolver{names: map[string]string{"U42": "bob"}}
	atts := mustAttachments(t, `[{"text": "ping <@U42>"}]`)
	got := RenderMessage(MessageContent{Attachments: atts}, resolver)
	assert.Equal(t, "ping @bob", got.Body)
}

func TestNormalizeWhitespace(t *testing.T) {
	assert.Equal(t, "a b c", normalizeWhitespace("  a   b\nc  "))
	assert.Equal(t, "", normalizeWhitespace("   \t\n  "))
	assert.Equal(t, "", normalizeWhitespace(""))
}
