package config

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type showOptions struct{}

func newShowCmd() *cobra.Command {
	opts := &showOptions{}

	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration status",
		Long: `Show credential configuration: backend, ref, and which keys are
present. Secret values are never displayed (§1.11).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(opts)
		},
	}
}

// showStatus is the non-secret view (§1.11 item 3): presence, backend, ref,
// workspace — never a token value, not even a masked prefix.
type showStatus struct {
	Ref              string `json:"credential_ref"`
	Backend          string `json:"backend"`
	BackendSource    string `json:"backend_source"`
	PassphraseSource string `json:"passphrase_source,omitempty"`
	Workspace        string `json:"workspace,omitempty"`
	BotToken         bool   `json:"bot_token_present"`
	UserToken        bool   `json:"user_token_present"`
}

func runShow(_ *showOptions) error {
	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}
	// OpenNoMigrate (not Open): config show is the §1.11 item 3 diagnostic.
	// It must remain usable during an unresolved §1.8 conflict so the user
	// can see which keys are where before remediating — running migration
	// first would fail it with ErrMigrationConflict and hide that state.
	st, err := keychain.OpenNoMigrate()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	backend, src := st.Backend()
	status := showStatus{
		Ref:           st.Ref(),
		Backend:       string(backend),
		BackendSource: string(src),
		Workspace:     cfg.Workspace,
		BotToken:      st.HasBotToken(),
		UserToken:     st.HasUserToken(),
	}
	if backend == credstore.BackendFile {
		svc, _, _ := credstore.ParseRef(st.Ref())
		status.PassphraseSource = keychain.PassphraseSource(svc)
	}

	if output.IsJSON() {
		return output.PrintJSON(status)
	}

	output.Printf("Credential ref: %s\n", status.Ref)
	output.Printf("Backend:        %s (%s)\n", status.Backend, status.BackendSource)
	if status.PassphraseSource != "" {
		output.Printf("Passphrase:     %s\n", status.PassphraseSource)
	}
	if status.Workspace != "" {
		output.Printf("Workspace:      %s\n", status.Workspace)
	}
	output.Printf("bot_token:      %s\n", presence(status.BotToken))
	output.Printf("user_token:     %s\n", presence(status.UserToken))
	if !status.BotToken && !status.UserToken {
		output.Println()
		output.Println("No credentials configured. Run 'slck init' or")
		output.Println("'slck set-credential --key bot_token --stdin'.")
	}
	return nil
}

func presence(ok bool) string {
	if ok {
		return "present"
	}
	return "not configured"
}
