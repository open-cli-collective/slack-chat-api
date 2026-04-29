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
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

var (
	// OutputFormat is the current output format (set by root command)
	OutputFormat Format = FormatText

	// NoColor disables colored output
	NoColor bool = false

	// Writer is where output goes (default os.Stdout, can be changed for testing)
	Writer io.Writer = os.Stdout
)

// JSON is deprecated: use IsJSON() instead.
// Kept for backward compatibility during transition.
var JSON bool

// IsJSON returns true if output format is JSON
func IsJSON() bool {
	return OutputFormat == FormatJSON || JSON
}

// PrintJSON outputs data as formatted JSON
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

func sanitizeSearchCell(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
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

// ValidFormats returns the list of valid output formats for flag validation
func ValidFormats() []string {
	return []string{string(FormatText), string(FormatJSON), string(FormatTable)}
}

// ParseFormat parses a string into a Format, returning an error if invalid
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "text", "":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "table":
		return FormatTable, nil
	default:
		return FormatText, fmt.Errorf("invalid output format %q: must be one of: text, json, table", s)
	}
}
