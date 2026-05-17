package config

import (
	"os"

	"github.com/spf13/cobra"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type clearOptions struct {
	all bool
}

func newClearCmd() *cobra.Command {
	opts := &clearOptions{}
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Remove stored credentials for the active profile",
		Long: `Remove the keyring credentials under the active credential_ref
(§1.7). Scope is the active profile only — other profiles and other CLIs are
untouched. With --all, also remove config.yml (return to a pre-init state).
Idempotent and non-interactive.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClear(opts)
		},
	}
	cmd.Flags().BoolVar(&opts.all, "all", false, "Also remove config.yml (pre-init state)")
	return cmd
}

func runClear(opts *clearOptions) error {
	// No-migration open: clear is the §1.8 conflict remediation ("run
	// config clear then re-run"). Running migration first would return
	// ErrMigrationConflict before we could delete the keyring entry.
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	removed, err := st.Clear()
	if err != nil {
		return err
	}
	if len(removed) == 0 {
		output.Printf("No keyring credentials under %s.\n", st.Ref())
	} else {
		for _, k := range removed {
			output.Printf("Removed %s from %s\n", k, st.Ref())
		}
	}

	if opts.all {
		// §1.7: --all returns the active profile to a pre-init state —
		// keyring bundle (above) + config file + caches + empty dirs.
		// slck has NO cache directory today (no command writes one); if
		// one is ever added it must also be removed here.
		p := appconfig.Path()
		switch err := os.Remove(p); {
		case err == nil:
			output.Printf("Removed %s\n", p)
		case os.IsNotExist(err):
			// idempotent
		default:
			return err
		}
	}
	return nil
}
