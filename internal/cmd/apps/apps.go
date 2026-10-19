// Package apps implements `slck apps`: managing Slack apps through their
// manifests (apps.manifest.*), authenticated with an app configuration
// token rather than the bot or user token.
package apps

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// exitError makes the root command exit with code instead of 1. A silent
// one prints nothing more, because the command already reported why.
type exitError struct {
	code   int
	err    error
	silent bool
}

func (e *exitError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}
func (e *exitError) Unwrap() error { return e.err }
func (e *exitError) ExitCode() int { return e.code }
func (e *exitError) Silent() bool  { return e.silent }

// cannotCompare is diff's exit status 2: the comparison could not be made,
// whether from a bad invocation, an unreadable file or a failed export.
func cannotCompare(err error) error {
	if err == nil {
		return nil
	}
	return &exitError{code: 2, err: err}
}

// appsHints map apps.manifest.* error codes to what to do about them.
var appsHints = map[string]string{
	"invalid_auth":     "Slack rejected the app configuration token even after rotating it. Generate a new pair at https://api.slack.com/apps and store it with 'slck set-credential --key app_config_token --stdin'.",
	"token_expired":    "Slack rejected the app configuration token even after rotating it. Generate a new pair at https://api.slack.com/apps and store it with 'slck set-credential --key app_config_token --stdin'.",
	"invalid_app_id":   "Check the app ID (it starts with A). The configuration token only reaches apps in its own workspace.",
	"invalid_app":      "Check the app ID (it starts with A). The configuration token only reaches apps in its own workspace.",
	"invalid_manifest": "Fix the problems listed above and run 'slck apps validate' again.",
}

// wrap names the operation and adds a hint for known apps.manifest.* errors.
func wrap(operation string, err error) error {
	var apiErr *client.AppsAPIError
	if errors.As(err, &apiErr) {
		if hint, ok := appsHints[apiErr.Code]; ok {
			return fmt.Errorf("%s: %w\nHint: %s", operation, err, hint)
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// quiet stops cobra from printing a runtime error and the usage text: once
// the arguments parsed, the root command prints the error alone, and a
// silent exitError (diff's "differs") prints nothing more.
func quiet(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
}

// NewCmd builds `slck apps`.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Manage Slack apps through their manifests",
		Long: `Manage Slack apps through their manifests (apps.manifest.* methods).

These commands authenticate with an app configuration token, not the bot or
user token. Generate one at https://api.slack.com/apps under "Your App
Configuration Tokens" and store the pair:

  <command that prints the pair as JSON> | slck set-credential --key app_config_token --stdin

The JSON is {"token": "xoxe.xoxp-...", "refresh_token": "xoxe-...", "exp": <unix seconds>}.
slck rotates the pair with tooling.tokens.rotate when the access token is
within five minutes of expiry or Slack rejects it, and saves the new pair to
the keyring before using it.

Installing an app and reading its bot token need a person in a browser;
these commands do not do either.`,
	}
	cmd.AddCommand(newGetCmd(), newValidateCmd(), newDiffCmd(), newCreateCmd(), newUpdateCmd(), newDeleteCmd())
	return cmd
}

func newGetCmd() *cobra.Command {
	var toFile string
	cmd := &cobra.Command{
		Use:   "get <app-id>",
		Short: "Print an app's manifest",
		Long: `Print an app's manifest as indented JSON with sorted keys, the form
the other apps commands read back. The manifest is the document itself, so
it is printed as JSON whatever --output says.`,
		Example: `  slck apps get A0123456789 > app-manifest.json
  slck apps get A0123456789 --to-file app-manifest.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runGet(newSession(), args[0], toFile)
		},
	}
	cmd.Flags().StringVar(&toFile, "to-file", "", "Write the manifest to this path instead of stdout")
	return cmd
}

func runGet(s *session, appID, toFile string) error {
	var manifest json.RawMessage
	err := s.do(func(token string) (err error) {
		manifest, err = s.api.ExportManifest(token, appID)
		return err
	})
	if err != nil {
		return wrap("export manifest of "+appID, err)
	}
	doc, err := indent(manifest)
	if err != nil {
		return err
	}
	if toFile == "" {
		_, err = output.Writer.Write(doc)
		return err
	}
	if err := os.WriteFile(toFile, doc, 0o644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	output.Printf("Wrote the manifest of %s to %s\n", appID, toFile)
	return nil
}

func newValidateCmd() *cobra.Command {
	var file, appID string
	cmd := &cobra.Command{
		Use:   "validate --file <manifest>",
		Short: "Check a manifest without changing anything",
		Long: `Check a manifest file with apps.manifest.validate. With --app, Slack
also checks it as an update to that app. Exits 0 when the manifest is valid
and non-zero, listing each problem, when it is not.`,
		Example: `  slck apps validate --file app-manifest.json
  slck apps validate --file app-manifest.yaml --app A0123456789`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runValidate(newSession(), file, appID)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "F", "", "Manifest file (.json, .yaml or .yml)")
	cmd.Flags().StringVar(&appID, "app", "", "Validate as an update to this app")
	return cmd
}

func runValidate(s *session, file, appID string) error {
	manifest, err := readManifest(file)
	if err != nil {
		return err
	}
	if err := s.do(func(token string) error { return s.api.ValidateManifest(token, manifest, appID) }); err != nil {
		return wrap("validate "+file, err)
	}
	output.Printf("Valid: %s\n", file)
	return nil
}

func newDiffCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "diff <app-id> --file <manifest>",
		Short: "Compare an app's live manifest with a file",
		Long: `Compare an app's live manifest with a manifest file, one line per
difference: "~" a value that differs, "-" a key only the live app has, "+" a
key only the file has. Paths are JSON pointers.

Exit status: 0 when they match, 1 when they differ, 2 when the comparison
could not be made.`,
		Example: `  slck apps diff A0123456789 --file app-manifest.json`,
		// A mistyped invocation must not exit 1, which means "differs".
		Args: func(cmd *cobra.Command, args []string) error {
			return cannotCompare(cobra.ExactArgs(1)(cmd, args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runDiff(newSession(), args[0], file)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "F", "", "Manifest file (.json, .yaml or .yml)")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return cannotCompare(err) })
	return cmd
}

func runDiff(s *session, appID, file string) error {
	manifest, err := readManifest(file)
	if err != nil {
		return cannotCompare(err)
	}
	var live json.RawMessage
	err = s.do(func(token string) (err error) {
		live, err = s.api.ExportManifest(token, appID)
		return err
	})
	if err != nil {
		return cannotCompare(wrap("export manifest of "+appID, err))
	}
	changes, err := diffManifests(live, manifest)
	if err != nil {
		return cannotCompare(err)
	}
	if len(changes) == 0 {
		output.Printf("No differences between %s and %s\n", appID, file)
		return nil
	}
	for _, c := range changes {
		output.Println(c.describe())
	}
	return &exitError{code: 1, silent: true}
}

func newCreateCmd() *cobra.Command {
	var file, credentialsFile string
	cmd := &cobra.Command{
		Use:   "create --file <manifest>",
		Short: "Create an app from a manifest",
		Long: `Create an app from a manifest file and print its app ID and the URL
that installs it.

Slack returns the new app's client secret, signing secret and verification
token once. They are never printed: --credentials-file writes them as JSON to
a new file readable only by you. Without it they are discarded, and the
Basic Information page of the app still shows them.`,
		Example: `  slck apps create --file app-manifest.json --credentials-file app-credentials.json`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runCreate(newSession(), file, credentialsFile)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "F", "", "Manifest file (.json, .yaml or .yml)")
	cmd.Flags().StringVar(&credentialsFile, "credentials-file", "", "Write the new app's credentials to this new file (mode 0600)")
	return cmd
}

func runCreate(s *session, file, credentialsFile string) error {
	manifest, err := readManifest(file)
	if err != nil {
		return err
	}
	// Open the credentials file first, so a bad path fails before an app
	// exists whose secrets could not be kept.
	var creds *os.File
	if credentialsFile != "" {
		creds, err = os.OpenFile(credentialsFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return fmt.Errorf("credentials file: %w", err)
		}
		defer func() { _ = creds.Close() }()
	}
	var app client.CreatedApp
	err = s.do(func(token string) (err error) {
		app, err = s.api.CreateApp(token, manifest)
		return err
	})
	if err != nil {
		if creds != nil {
			_ = creds.Close()
			_ = os.Remove(credentialsFile)
		}
		return wrap("create app from "+file, err)
	}
	if creds != nil {
		enc := json.NewEncoder(creds)
		enc.SetIndent("", "  ")
		if err := enc.Encode(app.Credentials); err != nil {
			return fmt.Errorf("created app %s but could not write its credentials: %w", app.AppID, err)
		}
	}
	output.Printf("Created app %s\n", app.AppID)
	if app.OAuthAuthorizeURL != "" {
		output.Printf("Install it: %s\n", app.OAuthAuthorizeURL)
	}
	if creds != nil {
		output.Printf("Credentials written to %s\n", credentialsFile)
	}
	return nil
}

func newUpdateCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "update <app-id> --file <manifest>",
		Short: "Replace an app's manifest",
		Long: `Replace an app's manifest with a manifest file. The file replaces the
whole manifest, so start from 'slck apps get'. Run 'slck apps diff' first to
see what changes.

When the change adds or removes scopes, Slack applies them only after the app
is reinstalled to the workspace, and the command says so.`,
		Example: `  slck apps diff A0123456789 --file app-manifest.json
  slck apps update A0123456789 --file app-manifest.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			return runUpdate(newSession(), args[0], file)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "F", "", "Manifest file (.json, .yaml or .yml)")
	return cmd
}

func runUpdate(s *session, appID, file string) error {
	manifest, err := readManifest(file)
	if err != nil {
		return err
	}
	var permissionsUpdated bool
	err = s.do(func(token string) (err error) {
		permissionsUpdated, err = s.api.UpdateApp(token, appID, manifest)
		return err
	})
	if err != nil {
		return wrap("update app "+appID, err)
	}
	output.Printf("Updated app %s from %s\n", appID, file)
	if permissionsUpdated {
		output.Println("Its scopes changed: reinstall the app to the workspace for them to take effect.")
	}
	return nil
}

type deleteOptions struct {
	force    bool
	stdin    io.Reader
	terminal bool
}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <app-id>",
		Short: "Permanently delete an app",
		Long: `Permanently delete an app with apps.manifest.delete. This cannot be
undone: the app, its tokens and its installations are gone. Asks for
confirmation unless --force is given, and refuses to run without --force
when stdin is not a terminal.`,
		Example: `  slck apps delete A0123456789`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			quiet(cmd)
			opts.stdin = os.Stdin
			opts.terminal = term.IsTerminal(int(os.Stdin.Fd()))
			return runDelete(newSession(), args[0], opts)
		},
	}
	cmd.Flags().BoolVar(&opts.force, "force", false, "Delete without asking for confirmation")
	return cmd
}

func runDelete(s *session, appID string, opts *deleteOptions) error {
	if !opts.force {
		if !opts.terminal {
			return errors.New("delete needs confirmation: stdin is not a terminal, so pass --force to delete without asking")
		}
		output.Printf("Permanently delete app %s? This cannot be undone. [y/N]: ", appID)
		answer := ""
		scanner := bufio.NewScanner(opts.stdin)
		if scanner.Scan() {
			answer = strings.ToLower(strings.TrimSpace(scanner.Text()))
		}
		if answer != "y" && answer != "yes" {
			output.Println("Cancelled.")
			return nil
		}
	}
	if err := s.do(func(token string) error { return s.api.DeleteApp(token, appID) }); err != nil {
		return wrap("delete app "+appID, err)
	}
	output.Printf("Deleted app %s\n", appID)
	return nil
}
