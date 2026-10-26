package config

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/apptoken"
	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type showOptions struct {
	json bool
}

func newShowCmd() *cobra.Command {
	opts := &showOptions{}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration status",
		Long: `Show credential configuration: backend, ref, and which keys are
present. Secret values are never displayed (§1.11).

With --json, emit a control-plane envelope (§2 carve-out). The global
--output flag is silently ignored when --json is set; the envelope is
the only output.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShow(opts)
		},
	}
	cmd.Flags().BoolVar(&opts.json, "json", false, "Emit a JSON control-plane envelope (§2 carve-out)")
	return cmd
}

// showStatus is the non-secret view (§1.11 item 3): presence, backend, ref,
// workspace — never a token value, not even a masked prefix.
type showStatus struct {
	Ref              string `json:"credential_ref"`
	Backend          string `json:"backend"`
	BackendSource    string `json:"backend_source"`
	KeyringBackend   string `json:"keyring_backend,omitempty"`
	PassphraseSource string `json:"passphrase_source,omitempty"`
	Workspace        string `json:"workspace,omitempty"`
	BotToken         bool   `json:"bot_token_present"`
	UserToken        bool   `json:"user_token_present"`
	AppConfigToken   bool   `json:"app_config_token_present"`
	AppToken         bool   `json:"app_token_present"`
	// AppConfigExpires is the access token's expiry (RFC 3339); slck rotates
	// the pair when it is used near or after this time.
	AppConfigExpires string `json:"app_config_token_expires,omitempty"`
	AppConfigTeamID  string `json:"app_config_token_team_id,omitempty"`
}

func runShow(opts *showOptions) error {
	cfg, err := appconfig.LoadForRuntime()
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
		Ref:            st.Ref(),
		Backend:        string(backend),
		BackendSource:  string(src),
		KeyringBackend: cfg.Keyring.Backend,
		Workspace:      cfg.Workspace,
		BotToken:       st.HasBotToken(),
		UserToken:      st.HasUserToken(),
		AppConfigToken: st.HasAppConfigToken(),
		AppToken:       st.HasAppToken(),
	}
	if status.AppConfigToken {
		if raw, err := st.AppConfigToken(); err == nil {
			if pair, err := apptoken.Parse(raw); err == nil {
				status.AppConfigExpires = pair.Expiry().UTC().Format(time.RFC3339)
				status.AppConfigTeamID = pair.TeamID
			}
		}
	}
	if backend == credstore.BackendFile {
		svc, _, _ := credstore.ParseRef(st.Ref())
		status.PassphraseSource = keychain.PassphraseSource(svc)
	}

	if opts.json {
		return output.PrintJSON(status)
	}

	output.Printf("Credential ref: %s\n", status.Ref)
	output.Printf("Backend:        %s (%s)\n", status.Backend, status.BackendSource)
	if status.KeyringBackend != "" {
		output.Printf("keyring.backend: %s (config.yml)\n", status.KeyringBackend)
	}
	if status.PassphraseSource != "" {
		output.Printf("Passphrase:     %s\n", status.PassphraseSource)
	}
	if status.Workspace != "" {
		output.Printf("Workspace:      %s\n", status.Workspace)
	}
	output.Printf("bot_token:      %s\n", presence(status.BotToken))
	output.Printf("user_token:     %s\n", presence(status.UserToken))
	appConfig := presence(status.AppConfigToken)
	if status.AppConfigExpires != "" {
		appConfig += ", access token expires " + status.AppConfigExpires
	}
	if status.AppConfigTeamID != "" {
		appConfig += ", team " + status.AppConfigTeamID
	}
	output.Printf("app_config_token: %s\n", appConfig)
	output.Printf("app_token:      %s\n", presence(status.AppToken))
	if !status.BotToken && !status.UserToken && !status.AppConfigToken && !status.AppToken {
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
