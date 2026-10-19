package apps

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// change is one difference between a live manifest and a file, at a JSON
// pointer.
type change struct {
	Pointer string
	Live    any
	File    any
	// Side is "live" or "file" when the key exists on only that side.
	Side string
}

// diffManifests lists the differences between two manifests, sorted by
// pointer. Objects are compared key by key; arrays and scalars are compared
// as whole values, so a reordered scope list counts as a change.
func diffManifests(live, file json.RawMessage) ([]change, error) {
	var l, f any
	if err := json.Unmarshal(live, &l); err != nil {
		return nil, fmt.Errorf("decode live manifest: %w", err)
	}
	if err := json.Unmarshal(file, &f); err != nil {
		return nil, fmt.Errorf("decode manifest file: %w", err)
	}
	var changes []change
	walk("", l, f, &changes)
	sort.Slice(changes, func(i, j int) bool { return changes[i].Pointer < changes[j].Pointer })
	return changes, nil
}

func walk(ptr string, live, file any, out *[]change) {
	lm, lok := live.(map[string]any)
	fm, fok := file.(map[string]any)
	if !lok || !fok {
		if !reflect.DeepEqual(live, file) {
			*out = append(*out, change{Pointer: ptr, Live: live, File: file})
		}
		return
	}
	keys := map[string]bool{}
	for k := range lm {
		keys[k] = true
	}
	for k := range fm {
		keys[k] = true
	}
	for k := range keys {
		child := ptr + "/" + escapePointer(k)
		lv, inLive := lm[k]
		fv, inFile := fm[k]
		switch {
		case !inFile:
			*out = append(*out, change{Pointer: child, Live: lv, Side: "live"})
		case !inLive:
			*out = append(*out, change{Pointer: child, File: fv, Side: "file"})
		default:
			walk(child, lv, fv, out)
		}
	}
}

// escapePointer escapes a key for a JSON pointer (RFC 6901).
func escapePointer(k string) string {
	return strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1")
}

// describe renders a change as one line: "~" changed, "-" only in the live
// app, "+" only in the file.
func (c change) describe() string {
	switch c.Side {
	case "live":
		return fmt.Sprintf("- %s: %s (live only)", c.Pointer, compact(c.Live))
	case "file":
		return fmt.Sprintf("+ %s: %s (file only)", c.Pointer, compact(c.File))
	default:
		return fmt.Sprintf("~ %s: %s -> %s", c.Pointer, compact(c.Live), compact(c.File))
	}
}

func compact(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
