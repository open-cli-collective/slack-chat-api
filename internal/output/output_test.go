package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestSearchTable(t *testing.T) {
	tests := []struct {
		name            string
		headers         []string
		rows            [][]string
		lastColMaxRunes int
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:    "basic pipe-delimited output",
			headers: []string{"REF", "USER", "TEXT"},
			rows: [][]string{
				{"C123/1.2", "alice", "hello world"},
			},
			lastColMaxRunes: 0,
			wantContains: []string{
				"REF | USER | TEXT\n",
				"C123/1.2 | alice | hello world\n",
			},
		},
		{
			name:    "newlines flattened in cells",
			headers: []string{"REF", "TEXT"},
			rows: [][]string{
				{"C1/1.0", "line one\nline two"},
			},
			lastColMaxRunes: 0,
			wantContains:    []string{"C1/1.0 | line one line two\n"},
			wantNotContains: []string{"line one\nline two"},
		},
		{
			name:    "literal pipe in cells replaced with broken bar",
			headers: []string{"REF", "TEXT"},
			rows: [][]string{
				{"C1/1.0", "a|b|c"},
			},
			lastColMaxRunes: 0,
			wantContains:    []string{"C1/1.0 | a¦b¦c\n"},
			wantNotContains: []string{"a|b|c"},
		},
		{
			name:    "only last column truncates",
			headers: []string{"REF", "USER", "TEXT"},
			rows: [][]string{
				{"C1234567890/1234567890.123456", "verylongusername", "this is the body that should get truncated when it grows past the cap"},
			},
			lastColMaxRunes: 20,
			wantContains: []string{
				"C1234567890/1234567890.123456 | verylongusername | ",
				"...",
			},
			wantNotContains: []string{"truncated when it grows past the cap"},
		},
		{
			name:            "row shorter than headers pads",
			headers:         []string{"A", "B", "C"},
			rows:            [][]string{{"x"}},
			lastColMaxRunes: 0,
			wantContains:    []string{"A | B | C\n", "x |  | \n"},
		},
		{
			name:            "row longer than headers drops extras",
			headers:         []string{"A", "B"},
			rows:            [][]string{{"x", "y", "z"}},
			lastColMaxRunes: 0,
			wantContains:    []string{"A | B\n", "x | y\n"},
			wantNotContains: []string{"z"},
		},
		{
			name:            "empty headers no panic",
			headers:         nil,
			rows:            [][]string{{"a"}},
			lastColMaxRunes: 0,
			wantContains:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			origWriter := Writer
			Writer = &buf
			defer func() { Writer = origWriter }()

			SearchTable(tt.headers, tt.rows, tt.lastColMaxRunes)

			out := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\n--- got ---\n%s", want, out)
				}
			}
			for _, dont := range tt.wantNotContains {
				if strings.Contains(out, dont) {
					t.Errorf("output should not contain %q\n--- got ---\n%s", dont, out)
				}
			}
		})
	}
}

func TestSearchTableTruncatesRunesNotBytes(t *testing.T) {
	var buf bytes.Buffer
	origWriter := Writer
	Writer = &buf
	defer func() { Writer = origWriter }()

	// 10 multi-byte runes, cap to 5 — should produce 2 runes + "..."
	SearchTable([]string{"REF", "TEXT"}, [][]string{{"C1/1.0", "你好你好你好你好你好"}}, 5)

	out := buf.String()
	if !strings.Contains(out, "你好...") {
		t.Errorf("expected rune-aware truncation, got: %q", out)
	}
}
