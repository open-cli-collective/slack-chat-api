package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Format represents the output format
type Format string

const (
	FormatText  Format = "text"
	FormatTable Format = "table"
)

var (
	// OutputFormat is the current output format (set by root command).
	// Closed-set {text, table} after #173; JSON is reserved for local
	// control-plane carve-outs (today only `config show --json`).
	OutputFormat Format = FormatText

	// NoColor disables colored output
	NoColor bool = false

	// Writer is where output goes (default os.Stdout, can be changed for testing)
	Writer io.Writer = os.Stdout

	// ErrWriter is where notices and warnings go (default os.Stderr, can be changed for testing)
	ErrWriter io.Writer = os.Stderr
)

// PrintJSON encodes data as indented JSON to Writer. It is a pure encoder
// — no global state, no migration splicing. Called by local control-plane
// `--json` carve-outs (e.g. `slck config show --json`).
func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Printf outputs a formatted string
func Printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(Writer, format, args...)
}

// Println outputs a line
func Println(args ...interface{}) {
	_, _ = fmt.Fprintln(Writer, args...)
}

// Table prints data in aligned columns with headers
func Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build format string
	formats := make([]string, len(headers))
	for i, w := range widths {
		formats[i] = fmt.Sprintf("%%-%ds", w)
	}
	format := strings.Join(formats, "  ") + "\n"

	// Print header
	headerArgs := make([]interface{}, len(headers))
	for i, h := range headers {
		headerArgs[i] = h
	}
	_, _ = fmt.Fprintf(Writer, format, headerArgs...)

	// Print separator
	total := 0
	for _, w := range widths {
		total += w
	}
	total += (len(widths) - 1) * 2 // account for spacing
	_, _ = fmt.Fprintln(Writer, strings.Repeat("-", total))

	// Print rows
	for _, row := range rows {
		rowArgs := make([]interface{}, len(headers))
		for i := range headers {
			if i < len(row) {
				rowArgs[i] = row[i]
			} else {
				rowArgs[i] = ""
			}
		}
		_, _ = fmt.Fprintf(Writer, format, rowArgs...)
	}
}

// SearchTable writes pipe-delimited, unpadded rows for agent-friendly
// search output: "h1 | h2 | h3" / "v1 | v2 | v3".
//
// Cell rules (callers pass raw strings only):
//   - internal newlines are collapsed to single spaces; carriage returns are stripped
//   - literal '|' is replaced with U+00A6 '¦' so a downstream parser can
//     split on " | " deterministically
//   - only the LAST column is truncated, rune-based, when lastColMaxRunes > 0;
//     all other columns (REF, IDs, dates, names) render in full
//
// If a row's length doesn't match headers, missing cells are padded with ""
// and extra cells are dropped — no panic.
func SearchTable(headers []string, rows [][]string, lastColMaxRunes int) {
	if len(headers) == 0 {
		return
	}

	cleanHeaders := make([]string, len(headers))
	for i, h := range headers {
		cleanHeaders[i] = sanitizeSearchCell(h)
	}
	_, _ = fmt.Fprintln(Writer, strings.Join(cleanHeaders, " | "))

	lastIdx := len(headers) - 1
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			var raw string
			if i < len(row) {
				raw = row[i]
			}
			c := sanitizeSearchCell(raw)
			if i == lastIdx && lastColMaxRunes > 0 {
				c = truncateRunes(c, lastColMaxRunes)
			}
			cells[i] = c
		}
		_, _ = fmt.Fprintln(Writer, strings.Join(cells, " | "))
	}
}

// SearchBlocks writes search results with the last column in full, for when
// the truncated SearchTable column hides what a result says.
//
// It prints the header line of every column but the last once, then one block
// per row: the row's other cells on one line (sanitized as in SearchTable),
// followed by the last cell with its line breaks kept and each line indented
// by two spaces. The two-space indent is what marks a body line: the body is
// written verbatim apart from dropping carriage returns and terminal control
// characters (see stripControl), so a literal "|" in the body is data, not a
// column separator. Blocks are separated by a blank line.
func SearchBlocks(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	lastIdx := len(headers) - 1
	cleanHeaders := make([]string, lastIdx)
	for i := 0; i < lastIdx; i++ {
		cleanHeaders[i] = sanitizeSearchCell(headers[i])
	}
	if lastIdx > 0 {
		_, _ = fmt.Fprintln(Writer, strings.Join(cleanHeaders, " | "))
	}

	for n, row := range rows {
		if n > 0 || lastIdx > 0 {
			_, _ = fmt.Fprintln(Writer)
		}
		meta := make([]string, lastIdx)
		for i := 0; i < lastIdx; i++ {
			var raw string
			if i < len(row) {
				raw = row[i]
			}
			meta[i] = sanitizeSearchCell(raw)
		}
		if lastIdx > 0 {
			_, _ = fmt.Fprintln(Writer, strings.Join(meta, " | "))
		}
		var body string
		if lastIdx < len(row) {
			body = stripControl(row[lastIdx])
		}
		for _, line := range strings.Split(body, "\n") {
			_, _ = fmt.Fprintln(Writer, "  "+line)
		}
	}
}

// stripControl drops carriage returns and every other control character
// except newline and tab, so text another workspace member wrote (a message,
// a display name, a channel name) cannot carry terminal escape sequences into
// the output. Every search cell goes through it.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, s)
}

func sanitizeSearchCell(s string) string {
	s = stripControl(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "¦")
	return s
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// HumanSize formats a byte count as "411 B", "12.3 KB", "4.5 MB", etc.
// Slack occasionally returns size=-1 for certain snippet types; clamp
// negatives to 0 rather than render "-1 B".
func HumanSize(n int64) string {
	if n < 0 {
		n = 0
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	case n < gb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/gb)
	}
}

// KeyValue prints a single key-value pair
func KeyValue(key string, value interface{}) {
	_, _ = fmt.Fprintf(Writer, "%-12s  %v\n", key+":", value)
}

// ValidFormats returns the list of valid output formats for flag validation.
// Closed-set {text, table} per #173; JSON is reserved for local control-plane
// carve-outs (today only `config show --json`).
func ValidFormats() []string {
	return []string{string(FormatText), string(FormatTable)}
}

// ParseFormat parses a string into a Format, returning an error if invalid.
// Closed-set policy: rejects `json` uniformly with `yaml` and any other
// non-{text,table} value.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "text", "":
		return FormatText, nil
	case "table":
		return FormatTable, nil
	default:
		return FormatText, fmt.Errorf("invalid output format %q: must be one of: text, table", s)
	}
}
