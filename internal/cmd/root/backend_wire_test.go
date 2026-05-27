package root

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cccredstore "github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
)

const serviceName = "slack-chat-api"

func resetState(t *testing.T) {
	t.Helper()
	keychain.SetBackendFlagOverride("", false)
	t.Cleanup(func() { keychain.SetBackendFlagOverride("", false) })
}

// newRootForWireTests builds an ad-hoc cobra root with just the
// --backend persistent flag and the WireBackendSelection PreRunE, so the
// tests can exercise the wiring contract without touching the package-
// level rootCmd singleton (state isolation).
func newRootForWireTests() *cobra.Command {
	root := &cobra.Command{
		Use: "root",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return WireBackendSelection(cmd)
		},
	}
	var backend string
	root.PersistentFlags().StringVar(&backend, cccredstore.BackendFlagName, "", cccredstore.BackendFlagUsage())
	return root
}

func newProbeCmd(name string) *cobra.Command {
	return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error { return nil }}
}

func TestWireBackendSelection_FlagSet(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	root := newRootForWireTests()
	root.AddCommand(newProbeCmd("probe"))
	root.SetArgs([]string{"probe", "--backend", "memory"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	v, set := keychain.GetBackendFlagOverride()
	if !set || v != "memory" {
		t.Errorf("override = (%q, %v); want (\"memory\", true)", v, set)
	}
}

func TestWireBackendSelection_FlagInvalid(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	root := newRootForWireTests()
	root.AddCommand(newProbeCmd("probe"))
	root.SetArgs([]string{"probe", "--backend", "bogus"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, cccredstore.ErrBackendNotImplemented) {
		t.Errorf("errors.Is(_, ErrBackendNotImplemented) = false; err=%v", err)
	}
	if !strings.Contains(err.Error(), "backend") {
		t.Errorf("error should mention --backend: %v", err)
	}
}

func TestWireBackendSelection_ConfigPassthrough(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "memory"); err != nil {
		t.Fatalf("BindBackendFlag: %v", err)
	}
	if opts.Backend != "" {
		t.Errorf("Backend = %q, want empty", opts.Backend)
	}
	if opts.ConfigBackend != cccredstore.BackendMemory {
		t.Errorf("ConfigBackend = %q, want %q", opts.ConfigBackend, cccredstore.BackendMemory)
	}
}

func TestWireBackendSelection_InvalidConfigDeferred(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")
	opts := &cccredstore.Options{}
	if err := cccredstore.BindBackendFlag(opts, "", false, "bogus"); err != nil {
		t.Fatalf("BindBackendFlag should NOT validate config: %v", err)
	}
	if string(opts.ConfigBackend) != "bogus" {
		t.Errorf("ConfigBackend = %q, want verbatim %q", opts.ConfigBackend, "bogus")
	}
}

// TestRealCommandTreeInheritsBackendFlag walks the real rootCmd tree
// returned by Command() and asserts every leaf inherits the --backend
// persistent flag by pointer identity. Regresses the cobra
// inherited-flag bug where a shadowed flag at a subcommand would silently
// replace the inherited one.
func TestRealCommandTreeInheritsBackendFlag(t *testing.T) {
	resetState(t)

	root := Command()
	rootFlag := root.PersistentFlags().Lookup(cccredstore.BackendFlagName)
	if rootFlag == nil {
		t.Fatal("root does not register --backend persistent flag")
	}

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, child := range c.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			got := child.Flag(cccredstore.BackendFlagName)
			if got == nil {
				t.Errorf("%s: --backend flag not inherited", child.CommandPath())
			} else if got != rootFlag {
				t.Errorf("%s: --backend pointer mismatch — shadowed?", child.CommandPath())
			}
			walk(child)
		}
	}
	walk(root)
}

// TestRootSingleton_PersistentPreRunE_WiresBackend exercises the REAL
// package-level rootCmd singleton — proves that after slck's existing
// output/as-user/as-bot validation, PersistentPreRunE actually invokes
// WireBackendSelection. Ad-hoc cobra trees in the other tests prove the
// helper works in isolation; this one proves it's actually called by
// the singleton's PreRunE.
//
// Cleanup is meticulous: the singleton's package globals (outputFormat,
// asUser, asBot, the --backend flag's value AND Changed bit, the
// keychain override) must be restored or existing root tests will see
// state leak.
func TestRootSingleton_PersistentPreRunE_WiresBackend(t *testing.T) {
	resetState(t)
	t.Setenv(cccredstore.BackendEnvVar(serviceName), "")

	root := Command()
	backendFlagP := root.PersistentFlags().Lookup(cccredstore.BackendFlagName)
	if backendFlagP == nil {
		t.Fatal("root singleton missing --backend persistent flag")
	}

	// Save state we are about to mutate.
	priorBackendVal := backendFlagP.Value.String()
	priorBackendChanged := backendFlagP.Changed
	priorOutputFormat := outputFormat
	priorAsUser := asUser
	priorAsBot := asBot

	t.Cleanup(func() {
		_ = backendFlagP.Value.Set(priorBackendVal)
		backendFlagP.Changed = priorBackendChanged
		outputFormat = priorOutputFormat
		asUser = priorAsUser
		asBot = priorAsBot
	})

	// Mutate as cobra parsing would.
	if err := backendFlagP.Value.Set("memory"); err != nil {
		t.Fatalf("set flag: %v", err)
	}
	backendFlagP.Changed = true
	outputFormat = "text" // valid for output.ParseFormat
	asUser = false
	asBot = false

	if err := root.PersistentPreRunE(root, nil); err != nil {
		t.Fatalf("PersistentPreRunE: %v", err)
	}
	v, set := keychain.GetBackendFlagOverride()
	if !set || v != "memory" {
		t.Errorf("singleton PreRunE failed to wire backend: override = (%q, %v); want (\"memory\", true)", v, set)
	}
}
