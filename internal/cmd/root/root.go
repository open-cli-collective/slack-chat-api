package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/canvas"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/channels"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/config"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/emoji"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/files"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/initcmd"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/me"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/messages"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/search"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/setcred"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/users"
	"github.com/open-cli-collective/slack-chat-api/internal/cmd/workspace"
	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/version"
)

var outputFormat string
var asUser bool
var asBot bool

// backendFlag is the pflag storage anchor for --backend. It is
// intentionally not read directly: WireBackendSelection consults the
// flag via cmd.Flag(...) so it can also observe the Changed bit. Do
// NOT replace those lookups with `if backendFlag != ""` — an explicit
// empty --backend ("") is distinct from "no flag supplied".
var backendFlag string

var rootCmd = &cobra.Command{
	Use:   "slck",
	Short: "A CLI tool for interacting with Slack",
	Long: `slck is a command-line interface for Slack.

It provides commands for managing channels, users, messages,
and other Slack workspace operations.

Configure credentials with:
  slck init
  slck set-credential --key bot_token --stdin   (e.g. via 'op read | ...')

Credentials are stored in the OS keyring; secret material (tokens) is
never read from environment variables or config files at runtime — env
vars are accepted only as ingress during setup. Non-secret backend
routing (the keyring backend selector) is the exception: it is taken
from --backend, SLACK_CHAT_API_KEYRING_BACKEND, or keyring.backend in
config.yml so users on headless / non-default platforms can opt in.`,
	Version: version.Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse and validate output format
		format, err := output.ParseFormat(outputFormat)
		if err != nil {
			return err
		}
		output.OutputFormat = format

		// If flag was provided to determine which token to use, set it in the client
		if asUser && asBot {
			return fmt.Errorf("cannot use both --as-user and --as-bot flags together")
		}
		if asBot {
			client.SetAsUser(false)
		}
		if asUser {
			client.SetAsUser(true)
		}

		return WireBackendSelection(cmd)
	},
}

// WireBackendSelection validates the user-supplied --backend flag (eagerly,
// at PersistentPreRunE time so `slck me --backend bogus` fails before
// touching the keyring) and records it on the keychain package-level
// override so openWith picks it up when it loads config. The cobra layer
// does NOT load config — config-side validation deliberately happens later
// at the openWith site, where the loaded keyring.backend value can be
// attributed back to config.yml in the resulting error.
//
// An invalid --backend errors here with the source prefix; an invalid
// keyring.backend in config.yml only errors on the first credential
// access (e.g. `slck me`) and `slck config show` surfaces the value
// verbatim for discoverability.
func WireBackendSelection(cmd *cobra.Command) error {
	flag := cmd.Flag(credstore.BackendFlagName)
	value := ""
	changed := false
	if flag != nil {
		value = flag.Value.String()
		changed = flag.Changed
	}

	// Validate the flag value only; result discarded — actual binding
	// happens in openWith via SetBackendFlagOverride. We pass an empty
	// config-side so this validates the flag independently of config.
	if err := credstore.BindBackendFlag(&credstore.Options{}, value, changed, ""); err != nil {
		return fmt.Errorf("--%s: %w", credstore.BackendFlagName, err)
	}

	keychain.SetBackendFlagOverride(value, changed)
	return nil
}

// Command returns the fully-configured root command (persistent --output
// flag, PersistentPreRunE, all subcommands). Exposed so tests — notably the
// §1.12 no-leak suite — can drive the real top-level command exactly as a
// user would, instead of constructing a subcommand without root's
// persistent flags (which would error out before the command ever runs).
func Command() *cobra.Command { return rootCmd }

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "Output format: text, json, or table")
	rootCmd.PersistentFlags().BoolVar(&output.NoColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().BoolVar(&asUser, "as-user", false, "Use user token")
	rootCmd.PersistentFlags().BoolVar(&asBot, "as-bot", false, "Use bot token")
	rootCmd.PersistentFlags().StringVar(&backendFlag, credstore.BackendFlagName, "", credstore.BackendFlagUsage())

	// Set custom version template to include commit and build date
	rootCmd.SetVersionTemplate("slck " + version.Info() + "\n")

	// Add subcommands
	rootCmd.AddCommand(canvas.NewCmd())
	rootCmd.AddCommand(channels.NewCmd())
	rootCmd.AddCommand(users.NewCmd())
	rootCmd.AddCommand(messages.NewCmd())
	rootCmd.AddCommand(search.NewCmd())
	rootCmd.AddCommand(workspace.NewCmd())
	rootCmd.AddCommand(me.NewCmd())
	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(emoji.NewCmd())
	rootCmd.AddCommand(files.NewCmd())
	rootCmd.AddCommand(initcmd.NewCmd())
	rootCmd.AddCommand(setcred.NewCmd())
}
