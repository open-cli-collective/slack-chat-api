package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Block is a top-level Slack Block Kit block. Only rich_text is rendered
// today; other types (section, image, context, etc.) are preserved in the
// JSON representation but yield empty output from RenderBlocks, letting
// callers fall back to the message's m.Text plain-text fallback.
type Block struct {
	Type     string            `json:"type"`
	BlockID  string            `json:"block_id,omitempty"`
	Elements []json.RawMessage `json:"elements,omitempty"`
}

// RichTextElement is one element inside a rich_text block. The zero value
// plus the non-nil Elements signal a container (rich_text_section,
// rich_text_list, rich_text_preformatted, rich_text_quote). Otherwise the
// element is an inline primitive (text, link, user, channel, emoji,
// broadcast, usergroup, date).
type RichTextElement struct {
	Type     string            `json:"type"`
	Elements []json.RawMessage `json:"elements,omitempty"`

	// rich_text_list fields (only populated when Type == "rich_text_list")
	ListStyle string `json:"-"` // "bullet" | "ordered" — populated from polymorphic "style"
	Indent    int    `json:"indent,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	Border    int    `json:"border,omitempty"`

	// inline element fields
	Text        string `json:"text,omitempty"`
	URL         string `json:"url,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	ChannelID   string `json:"channel_id,omitempty"`
	Name        string `json:"name,omitempty"`      // emoji name
	Range       string `json:"range,omitempty"`     // broadcast: here|channel|everyone
	Timestamp   int64  `json:"timestamp,omitempty"` // date element (unix seconds)
	Format      string `json:"format,omitempty"`    // date format (not interpreted — Fallback/RFC3339 used instead)
	Fallback    string `json:"fallback,omitempty"`  // date fallback text
	UsergroupID string `json:"usergroup_id,omitempty"`

	TextStyle *RichTextStyle `json:"-"` // populated from polymorphic "style"
}

// RichTextStyle captures inline text formatting flags on a "text" element.
type RichTextStyle struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Code   bool `json:"code,omitempty"`
	Strike bool `json:"strike,omitempty"`
}

// UnmarshalJSON handles the "style" field's polymorphism: rich_text_list
// uses a string ("bullet"|"ordered"), inline text uses an object of
// boolean flags. Everything else is unmarshaled via the standard rules
// after stripping "style" out to avoid the default unmarshaler choking.
func (e *RichTextElement) UnmarshalJSON(data []byte) error {
	type alias RichTextElement
	aux := struct {
		*alias
		Style json.RawMessage `json:"style,omitempty"`
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Style) == 0 {
		return nil
	}
	trimmed := strings.TrimLeft(string(aux.Style), " \t\n\r")
	if strings.HasPrefix(trimmed, `"`) {
		var s string
		if err := json.Unmarshal(aux.Style, &s); err != nil {
			return err
		}
		e.ListStyle = s
		return nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var ts RichTextStyle
		if err := json.Unmarshal(aux.Style, &ts); err != nil {
			return err
		}
		e.TextStyle = &ts
	}
	return nil
}

// MarshalJSON round-trips the polymorphic style field so --output json
// preserves whichever variant was on the wire.
func (e RichTextElement) MarshalJSON() ([]byte, error) {
	type alias RichTextElement
	aux := struct {
		alias
		Style interface{} `json:"style,omitempty"`
	}{alias: alias(e)}
	switch {
	case e.ListStyle != "":
		aux.Style = e.ListStyle
	case e.TextStyle != nil:
		aux.Style = e.TextStyle
	}
	return json.Marshal(aux)
}

// RenderBlocks walks top-level blocks and returns plain text suitable for
// slck's text output. Returns "" if blocks is empty or no block yields
// output — callers can use that signal to fall back to m.Text. Safe to
// call with a nil resolver.
func RenderBlocks(blocks []Block, resolver *UserResolver) string {
	if len(blocks) == 0 {
		return ""
	}
	var out strings.Builder
	for _, b := range blocks {
		if b.Type != "rich_text" {
			// Non-rich_text blocks (section, image, context, etc.) are
			// dropped so the caller falls back to m.Text, which Slack
			// populates as a plain-text equivalent for these cases.
			continue
		}
		out.WriteString(renderRichText(b.Elements, resolver, 0))
	}
	return strings.TrimRight(out.String(), "\n")
}

// renderRichText walks the sub-elements of a rich_text block.
func renderRichText(raws []json.RawMessage, resolver *UserResolver, baseIndent int) string {
	var out strings.Builder
	for _, raw := range raws {
		var el RichTextElement
		if err := json.Unmarshal(raw, &el); err != nil {
			continue
		}
		switch el.Type {
		case "rich_text_section":
			out.WriteString(renderInline(el.Elements, resolver))
			out.WriteString("\n")
		case "rich_text_preformatted":
			out.WriteString("```\n")
			out.WriteString(renderInline(el.Elements, resolver))
			out.WriteString("\n```\n")
		case "rich_text_quote":
			body := renderInline(el.Elements, resolver)
			for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
				out.WriteString("> ")
				out.WriteString(line)
				out.WriteString("\n")
			}
		case "rich_text_list":
			indent := strings.Repeat("  ", baseIndent+el.Indent)
			start := el.Offset
			for i, itemRaw := range el.Elements {
				var item RichTextElement
				if err := json.Unmarshal(itemRaw, &item); err != nil {
					continue
				}
				// List items are rich_text_section elements
				body := renderInline(item.Elements, resolver)
				out.WriteString(indent)
				if el.ListStyle == "ordered" {
					fmt.Fprintf(&out, "%d. ", start+i+1)
				} else {
					out.WriteString("- ")
				}
				out.WriteString(body)
				out.WriteString("\n")
			}
		default:
			// Unknown sub-block type — drop silently.
		}
	}
	return out.String()
}

// renderInline walks the inline elements of a rich_text_section.
func renderInline(raws []json.RawMessage, resolver *UserResolver) string {
	var out strings.Builder
	for _, raw := range raws {
		var el RichTextElement
		if err := json.Unmarshal(raw, &el); err != nil {
			continue
		}
		switch el.Type {
		case "text":
			out.WriteString(applyTextStyle(el.Text, el.TextStyle))
		case "link":
			if el.Text != "" {
				fmt.Fprintf(&out, "[%s](%s)", el.Text, el.URL)
			} else {
				fmt.Fprintf(&out, "<%s>", el.URL)
			}
		case "user":
			name := resolver.Resolve(el.UserID)
			fmt.Fprintf(&out, "@%s", name)
		case "channel":
			fmt.Fprintf(&out, "<#%s>", el.ChannelID)
		case "emoji":
			fmt.Fprintf(&out, ":%s:", el.Name)
		case "broadcast":
			fmt.Fprintf(&out, "@%s", el.Range)
		case "usergroup":
			fmt.Fprintf(&out, "<!subteam^%s>", el.UsergroupID)
		case "date":
			if el.Fallback != "" {
				out.WriteString(el.Fallback)
			} else if el.Timestamp > 0 {
				out.WriteString(time.Unix(el.Timestamp, 0).UTC().Format(time.RFC3339))
			}
			// Slack's date-format token language ({date_num}, {date}, etc.)
			// is intentionally not interpreted — Format is read for JSON
			// round-tripping but ignored during rendering.
		default:
			// Unknown inline element — drop silently so fall-back to
			// m.Text can win if every element is unknown.
		}
	}
	return out.String()
}

func applyTextStyle(s string, style *RichTextStyle) string {
	if style == nil || s == "" {
		return s
	}
	if style.Code {
		s = "`" + s + "`"
	}
	if style.Bold {
		s = "**" + s + "**"
	}
	if style.Italic {
		s = "*" + s + "*"
	}
	if style.Strike {
		s = "~~" + s + "~~"
	}
	return s
}
