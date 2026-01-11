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
