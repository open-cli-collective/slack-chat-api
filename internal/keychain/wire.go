package keychain

import "sync"

// Backend-flag override lifecycle:
//
//   - Written by root.WireBackendSelection at cobra PersistentPreRunE time,
//     once per Execute(), from the parsed --backend flag.
//   - Read by openWith on every keychain.Open* call.
//   - Zero values (("", false)) mean "no --backend supplied" and openWith
//     falls through to env > config.Keyring.Backend > auto-detect via
//     credstore.BindBackendFlag.
//   - RWMutex defends against tests that read/write from parallel
//     goroutines; cobra itself dispatches PreRunE on the main goroutine
//     before any RunE, so there is no production race.
//
// Any new caller of openWith must run after PreRunE has fired (or
// explicitly call SetBackendFlagOverride first). Tests that mutate the
// override should defer a reset to keep state isolated.
var (
	backendMu         sync.RWMutex
	backendFlagValue  string
	backendFlagWasSet bool
)

// SetBackendFlagOverride records the user-supplied --backend flag for
// the next openWith call. Called by root.WireBackendSelection at
// PersistentPreRunE time. flagSet matches cobra's pflag.Flag.Changed —
// true when the user passed --backend on the command line, regardless
// of whether the value is empty.
func SetBackendFlagOverride(value string, flagSet bool) {
	backendMu.Lock()
	defer backendMu.Unlock()
	backendFlagValue = value
	backendFlagWasSet = flagSet
}

// GetBackendFlagOverride returns the current override.
func GetBackendFlagOverride() (value string, flagSet bool) {
	backendMu.RLock()
	defer backendMu.RUnlock()
	return backendFlagValue, backendFlagWasSet
}
