package client

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderBlocks_NilAndEmpty(t *testing.T) {
	assert.Equal(t, "", RenderBlocks(nil, nil))
	assert.Equal(t, "", RenderBlocks([]Block{}, nil))
}

func TestRenderBlocks_NilResolverDoesNotPanic(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_section", "elements": [
				{"type": "text", "text": "hi "},
				{"type": "user", "user_id": "U123"}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	// Resolver nil → user mentions fall back to raw ID prefixed with @.
	assert.Equal(t, "hi @U123", got)
}

func TestRenderBlocks_RichTextSection(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_section", "elements": [
				{"type": "text", "text": "hello "},
				{"type": "text", "text": "world", "style": {"bold": true}}
			]}
		]
	}]`)
	assert.Equal(t, "hello **world**", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_TextStyleCombinations(t *testing.T) {
	tests := []struct {
		name, rawStyle, expected string
	}{
		{"bold", `{"bold":true}`, "**x**"},
		{"italic", `{"italic":true}`, "*x*"},
		{"code", `{"code":true}`, "`x`"},
		{"strike", `{"strike":true}`, "~~x~~"},
		{"bold_italic", `{"bold":true,"italic":true}`, "***x***"},
		{"code_bold", `{"code":true,"bold":true}`, "**`x`**"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := `[{"type":"rich_text","elements":[{"type":"rich_text_section","elements":[{"type":"text","text":"x","style":` + tt.rawStyle + `}]}]}]`
			blocks := mustBlocks(t, raw)
			assert.Equal(t, tt.expected, RenderBlocks(blocks, nil))
		})
	}
}

func TestRenderBlocks_BulletList(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_list", "style": "bullet", "elements": [
				{"type": "rich_text_section", "elements": [{"type": "text", "text": "first"}]},
				{"type": "rich_text_section", "elements": [{"type": "text", "text": "second"}]}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Contains(t, got, "- first")
	assert.Contains(t, got, "- second")
}

func TestRenderBlocks_OrderedListHonorsOffset(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_list", "style": "ordered", "offset": 4, "elements": [
				{"type": "rich_text_section", "elements": [{"type": "text", "text": "apple"}]},
				{"type": "rich_text_section", "elements": [{"type": "text", "text": "banana"}]}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Contains(t, got, "5. apple")
	assert.Contains(t, got, "6. banana")
}

func TestRenderBlocks_NestedListIndent(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_list", "style": "bullet", "indent": 1, "elements": [
				{"type": "rich_text_section", "elements": [{"type": "text", "text": "child"}]}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Contains(t, got, "  - child", "expected two-space indent before bullet")
}

func TestRenderBlocks_Preformatted(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_preformatted", "elements": [
				{"type": "text", "text": "code block body"}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Contains(t, got, "```")
	assert.Contains(t, got, "code block body")
}

func TestRenderBlocks_Quote(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_quote", "elements": [
				{"type": "text", "text": "line one\nline two"}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	for _, want := range []string{"> line one", "> line two"} {
		assert.Contains(t, got, want)
	}
}

func TestRenderBlocks_InlineElementTypes(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_section", "elements": [
				{"type": "text", "text": "pre "},
				{"type": "link", "url": "https://example.com", "text": "site"},
				{"type": "text", "text": " "},
				{"type": "link", "url": "https://bare.example.com"},
				{"type": "text", "text": " "},
				{"type": "channel", "channel_id": "C123"},
				{"type": "text", "text": " "},
				{"type": "emoji", "name": "wave"},
				{"type": "text", "text": " "},
				{"type": "broadcast", "range": "here"},
				{"type": "text", "text": " "},
				{"type": "usergroup", "usergroup_id": "S01"}
			]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Contains(t, got, "[site](https://example.com)")
	assert.Contains(t, got, "<https://bare.example.com>")
	assert.Contains(t, got, "<#C123>")
	assert.Contains(t, got, ":wave:")
	assert.Contains(t, got, "@here")
	assert.Contains(t, got, "<!subteam^S01>")
}

func TestRenderBlocks_DateUsesFallbackOrRFC3339(t *testing.T) {
	t.Run("fallback wins when present", func(t *testing.T) {
		blocks := mustBlocks(t, `[{
			"type": "rich_text",
			"elements": [
				{"type": "rich_text_section", "elements": [
					{"type": "date", "timestamp": 1700000000, "format": "{date_num}", "fallback": "Nov 14, 2023"}
				]}
			]
		}]`)
		assert.Equal(t, "Nov 14, 2023", RenderBlocks(blocks, nil))
	})

	t.Run("RFC3339 when no fallback", func(t *testing.T) {
		blocks := mustBlocks(t, `[{
			"type": "rich_text",
			"elements": [
				{"type": "rich_text_section", "elements": [
					{"type": "date", "timestamp": 1700000000, "format": "{date_num}"}
				]}
			]
		}]`)
		got := RenderBlocks(blocks, nil)
		// RFC3339 of 1700000000 UTC
		assert.Contains(t, got, "2023-11-14T")
	})
}

func TestRenderBlocks_UnknownBlockTypeDropped(t *testing.T) {
	// A non-rich_text block should be silently dropped so callers can
	// fall back to m.Text, which Slack populates for such messages.
	blocks := mustBlocks(t, `[{"type":"section","block_id":"B1"}]`)
	assert.Equal(t, "", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_UnknownSubBlockDropped(t *testing.T) {
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [{"type": "rich_text_future_thing", "elements": []}]
	}]`)
	assert.Equal(t, "", RenderBlocks(blocks, nil))
}

func TestRenderBlocks_UnknownInlineElementDropped(t *testing.T) {
	// Unknown inline elements yield empty output — if every element is
	// unknown, overall render is empty, letting fallback-to-text win.
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [{"type": "rich_text_section", "elements": [
			{"type": "unknown_inline_type", "text": "ignored"}
		]}]
	}]`)
	assert.Equal(t, "", RenderBlocks(blocks, nil))
}

func TestRichTextElement_StylePolymorphism(t *testing.T) {
	t.Run("list style is a string", func(t *testing.T) {
		var el RichTextElement
		require.NoError(t, json.Unmarshal([]byte(`{"type":"rich_text_list","style":"ordered"}`), &el))
		assert.Equal(t, "ordered", el.ListStyle)
		assert.Nil(t, el.TextStyle)
	})

	t.Run("text style is an object", func(t *testing.T) {
		var el RichTextElement
		require.NoError(t, json.Unmarshal([]byte(`{"type":"text","text":"x","style":{"bold":true,"italic":true}}`), &el))
		assert.Equal(t, "", el.ListStyle)
		require.NotNil(t, el.TextStyle)
		assert.True(t, el.TextStyle.Bold)
		assert.True(t, el.TextStyle.Italic)
	})

	t.Run("missing style is fine", func(t *testing.T) {
		var el RichTextElement
		require.NoError(t, json.Unmarshal([]byte(`{"type":"text","text":"x"}`), &el))
		assert.Equal(t, "", el.ListStyle)
		assert.Nil(t, el.TextStyle)
	})
}

func TestRenderBlocks_MultilineSectionsSeparatedByNewlines(t *testing.T) {
	// Two sections → two rendered lines, joined by newline (but trailing
	// newline on the final output is trimmed).
	blocks := mustBlocks(t, `[{
		"type": "rich_text",
		"elements": [
			{"type": "rich_text_section", "elements": [{"type": "text", "text": "line1"}]},
			{"type": "rich_text_section", "elements": [{"type": "text", "text": "line2"}]}
		]
	}]`)
	got := RenderBlocks(blocks, nil)
	assert.Equal(t, "line1\nline2", got)
	assert.False(t, strings.HasSuffix(got, "\n"), "trailing newline should be trimmed")
}

// mustBlocks unmarshals a JSON array into []Block or fails the test.
func mustBlocks(t *testing.T, raw string) []Block {
	t.Helper()
	var blocks []Block
	require.NoError(t, json.Unmarshal([]byte(raw), &blocks))
	return blocks
}
