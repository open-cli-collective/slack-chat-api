package apps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// readManifest reads a manifest file as JSON, or as YAML when its extension
// is .yaml or .yml, and returns it as compact JSON for the API.
func readManifest(path string) (json.RawMessage, error) {
	if path == "" {
		return nil, fmt.Errorf("--file is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var doc any
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse manifest %s as YAML: %w", path, err)
		}
	default:
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse manifest %s as JSON: %w", path, err)
		}
	}
	if _, ok := doc.(map[string]any); !ok {
		return nil, fmt.Errorf("manifest %s must be an object", path)
	}
	return json.Marshal(doc)
}

// indent renders a manifest for people and for files: two-space indented
// JSON with sorted keys and a trailing newline, so an exported manifest
// diffs cleanly in version control.
func indent(manifest json.RawMessage) ([]byte, error) {
	var doc any
	if err := json.Unmarshal(manifest, &doc); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
