package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type migBlock struct {
	Version int      `json:"version"`
	Changes []string `json:"changes"`
}

func capture(t *testing.T, data interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	orig := Writer
	Writer = &buf
	t.Cleanup(func() { Writer = orig })
	if err := PrintJSON(data); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	return buf.String()
}

func TestPrintJSON_ObjectMergesMigrationFirst(t *testing.T) {
	ResetMigration()
	RecordMigration(migBlock{Version: 1, Changes: []string{"bot_token"}})
	out := capture(t, map[string]string{"result": "ok"})

	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("not valid JSON object: %v\n%s", err, out)
	}
	if _, ok := m["_migration"]; !ok {
		t.Fatalf("_migration not merged: %s", out)
	}
	if _, ok := m["result"]; !ok {
		t.Fatalf("original field lost: %s", out)
	}
	// Consume-once: a second emit has no _migration.
	out2 := capture(t, map[string]string{"result": "ok"})
	if strings.Contains(out2, "_migration") {
		t.Fatalf("_migration leaked into a later response: %s", out2)
	}
}

func TestPrintJSON_NonObjectWrapped(t *testing.T) {
	ResetMigration()
	RecordMigration(migBlock{Version: 1, Changes: []string{"user_token"}})
	out := capture(t, []string{"a", "b"})

	var w struct {
		Migration migBlock          `json:"_migration"`
		Data      []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		t.Fatalf("array not wrapped into object: %v\n%s", err, out)
	}
	if len(w.Data) != 2 || w.Migration.Version != 1 {
		t.Fatalf("wrap shape wrong: %s", out)
	}
}

func TestPrintJSON_EmptyObjectStillValid(t *testing.T) {
	ResetMigration()
	RecordMigration(migBlock{Version: 1})
	out := capture(t, map[string]string{})
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &m); err != nil || len(m) != 1 {
		t.Fatalf("empty-object splice invalid: %v\n%s", err, out)
	}
}

func TestPrintJSON_NoMigrationUnchanged(t *testing.T) {
	ResetMigration()
	out := capture(t, map[string]string{"k": "v"})
	if strings.Contains(out, "_migration") {
		t.Fatalf("unexpected _migration with none recorded: %s", out)
	}
}
