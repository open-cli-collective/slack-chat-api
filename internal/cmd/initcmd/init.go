package initcmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	appconfig "github.com/open-cli-collective/slack-chat-api/internal/config"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// initOptions: tokens are NOT accepted as flag/positional values (§1.5/§1.12).
// They arrive only via a named env var, single-secret stdin, or an
// interactive prompt — all valid *ingress* mechanisms during setup.
type initOptions struct {
	botEnv    string // --bot-token-from-env NAME
	userEnv   string // --user-token-from-env NAME
	botStdin  bool   // --bot-token-stdin (single-secret stdin)
	noVerify  bool
	overwrite bool // --overwrite: resolve a §1.8 legacy/keyring conflict

	stdin     io.Reader                                  // test seam (prompts / --bot-token-stdin)
	newClient func(baseURL, token string) *client.Client // test seam
}

// NewCmd creates the init command.
func NewCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard",
		Long: `Set up slck.

Tokens are read from a named environment variable, single-secret stdin, or
an interactive prompt — never from a flag value (which would leak via shell
history / process listings, §1.5/§1.12). Examples:

  slck init --bot-token-from-env SLACK_BOT_TOKEN --user-token-from-env SLACK_USER_TOKEN
  op read 'op://Vault/Slack/bot' | slck init --bot-token-stdin
  slck init                       # interactive`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.botEnv, "bot-token-from-env", "", "Read the bot token from this env var")
	cmd.Flags().StringVar(&opts.userEnv, "user-token-from-env", "", "Read the user token from this env var")
	cmd.Flags().BoolVar(&opts.botStdin, "bot-token-stdin", false, "Read the bot token from stdin")
	cmd.Flags().BoolVar(&opts.noVerify, "no-verify", false, "Skip token verification")
	cmd.Flags().BoolVar(&opts.overwrite, "overwrite", false, "Resolve a legacy/keyring migration conflict by forcing the legacy value")
	return cmd
}

func (o *initOptions) reader() io.Reader {
	if o.stdin != nil {
		return o.stdin
	}
	return os.Stdin
}

func (o *initOptions) makeClient(token string) *client.Client {
	if o.newClient != nil {
		return o.newClient("", token)
	}
	return client.NewWithConfig("https://slack.com/api", token, nil)
}

func runInit(opts *initOptions) error {
	output.Println("Slack CLI Setup")
	output.Println()

	// Opening the store runs the one-time legacy migration (§1.8). With
	// --overwrite a legacy/keyring conflict is resolved by forcing legacy.
	open := keychain.Open
	if opts.overwrite {
		open = keychain.OpenForMigrationOverwrite
	}
	st, err := open()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	if (st.HasBotToken() || st.HasUserToken()) && !opts.overwrite && opts.interactive() {
		if !promptYesNo(opts.reader(), "Credentials already exist. Overwrite?", false) {
			output.Println("Setup cancelled.")
			return nil
		}
	}

	botToken, err := opts.resolveBot()
	if err != nil {
		return err
	}
	userToken, err := opts.resolveUser()
	if err != nil {
		return err
	}
	if botToken == "" && userToken == "" {
		output.Println("No tokens provided. Setup cancelled.")
		return nil
	}

	workspace := ""

	if botToken != "" {
		if t := keychain.DetectTokenType(botToken); t != "bot" {
			return fmt.Errorf("expected bot token (xoxb-*), got %s token", t)
		}
		if !opts.noVerify {
			info, err := opts.verify(botToken, "Bot")
			if err != nil {
				return fmt.Errorf("bot token verification failed: %w", err)
			}
			workspace = info.TeamID
		}
		if err := st.SetBotToken(botToken); err != nil {
			return err
		}
		output.Println("Bot token saved.")
	}

	if userToken != "" {
		if t := keychain.DetectTokenType(userToken); t != "user" {
			return fmt.Errorf("expected user token (xoxp-*), got %s token", t)
		}
		if !opts.noVerify {
			info, err := opts.verify(userToken, "User")
			if err != nil {
				return fmt.Errorf("user token verification failed: %w", err)
			}
			if workspace == "" {
				workspace = info.TeamID
			}
		}
		if err := st.SetUserToken(userToken); err != nil {
			return err
		}
		output.Println("User token saved.")
	}

	// Persist non-secret config (credential_ref + workspace, §1.2/§2.4).
	cfg, err := appconfig.Load()
	if err != nil {
		return err
	}
	if workspace != "" {
		cfg.Workspace = workspace
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	output.Println()
	output.Println("Configuration saved. Try it out:")
	if botToken != "" {
		output.Println("  slck channels list")
	}
	if userToken != "" {
		output.Println("  slck search messages \"hello\"")
	}
	return nil
}

// interactive reports whether init will prompt (no ingress flags given).
func (o *initOptions) interactive() bool {
	return o.botEnv == "" && o.userEnv == "" && !o.botStdin
}

func (o *initOptions) verify(token, label string) (*client.AuthTestResponse, error) {
	output.Printf("Verifying %s token...\n", label)
	info, err := o.makeClient(token).AuthTest()
	if err != nil {
		return nil, err
	}
	output.Printf("  Connected to workspace: %s\n", info.Team)
	return info, nil
}

func (o *initOptions) resolveBot() (string, error) {
	switch {
	case o.botEnv != "":
		return os.Getenv(o.botEnv), nil
	case o.botStdin:
		b, err := io.ReadAll(o.reader())
		if err != nil {
			return "", fmt.Errorf("read bot token from stdin: %w", err)
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	default:
		return promptToken(o.reader(), "Bot Token (xoxb-...)")
	}
}

func (o *initOptions) resolveUser() (string, error) {
	if o.userEnv != "" {
		return os.Getenv(o.userEnv), nil
	}
	if !o.interactive() {
		return "", nil // non-interactive run only sets what was supplied
	}
	if !promptYesNo(o.reader(), "Add a user token as well? (needed for search)", false) {
		return "", nil
	}
	return promptToken(o.reader(), "User Token (xoxp-...)")
}

func promptToken(reader io.Reader, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func promptYesNo(reader io.Reader, prompt string, defaultYes bool) bool {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	fmt.Printf("%s%s", prompt, suffix)

	scanner := bufio.NewScanner(reader)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "" {
			return defaultYes
		}
		return answer == "y" || answer == "yes"
	}
	return defaultYes
}
