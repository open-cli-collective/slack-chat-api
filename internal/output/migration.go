package output

import (
	"bytes"
	"encoding/json"
	"sync"
)

// The §1.8 machine-readable migration signal. When the one-time legacy
// migration runs, it records a block here; the next JSON emit splices it in
// and clears it (run-scoped, consume-once) so it appears exactly once and
// never leaks into a later response or a parallel test.
//
// Policy (Codex-reviewed, template for B2/B3):
//   - object responses  -> "_migration" merged as the first top-level field,
//     original fields preserved verbatim and in order.
//   - non-object responses (slck emits arrays for list/history endpoints)
//     -> wrapped as {"_migration": ..., "data": <original>}.
var (
	migMu      sync.Mutex
	migPending []byte // marshaled migration block value, or nil if none
)

// RecordMigration stores the §1.8 block (anything that marshals to the
// `_migration` *value*, e.g. credstore.MigrationBlock). nil/empty changes
// must not call this — absence means "no migration this run".
func RecordMigration(block interface{}) {
	b, err := json.Marshal(block)
	if err != nil {
		return // a marshal failure here must not break the actual command
	}
	migMu.Lock()
	migPending = b
	migMu.Unlock()
}

// takeMigration returns the pending block and clears it (consume-once).
func takeMigration() []byte {
	migMu.Lock()
	defer migMu.Unlock()
	b := migPending
	migPending = nil
	return b
}

// ResetMigration drops any pending block without emitting it. Test hook so
// one test's recorded migration can never bleed into another.
func ResetMigration() {
	migMu.Lock()
	migPending = nil
	migMu.Unlock()
}

// spliceMigration applies the object-merge / non-object-wrap policy to an
// already-marshaled response body, returning compact JSON.
func spliceMigration(body, mig []byte) []byte {
	t := bytes.TrimSpace(body)
	if len(t) > 0 && t[0] == '{' && t[len(t)-1] == '}' {
		inner := bytes.TrimSpace(t[1 : len(t)-1])
		if len(inner) == 0 {
			return []byte(`{"_migration":` + string(mig) + `}`)
		}
		return []byte(`{"_migration":` + string(mig) + `,` + string(inner) + `}`)
	}
	return []byte(`{"_migration":` + string(mig) + `,"data":` + string(t) + `}`)
}
