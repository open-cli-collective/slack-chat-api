// Package setcred implements `slck set-credential` — the low-level,
// single-secret, scriptable credential ingress (§1.5.2). It accepts the
// value only via stdin or a named env var, never as a flag/positional
// value, and only for allowed keys.
package setcred

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/keychain"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type options struct {
	ref     string
	key     string
	stdin   bool
	fromEnv string
	in      io.Reader // test seam
}

// NewCmd builds the `slck set-credential` command.
func NewCmd() *cobra.Command {
	opts := &options{}
	cmd := &cobra.Command{
		Use:   "set-credential --key <bot_token|user_token> (--stdin | --from-env NAME)",
		Short: "Store one credential in the keyring (stdin or env ingress)",
		Long: `Store a single secret in the OS keyring (§1.5.2).

The value is read ONLY from stdin (--stdin) or a named environment variable
(--from-env NAME) — never from a flag or positional argument. Only the
keys 'bot_token' and 'user_token' are accepted.

  op read 'op://Vault/Slack/bot_token' | slck set-credential --key bot_token --stdin
  slck set-credential --key user_token --from-env SLACK_USER_TOKEN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(opts)
		},
	}
	cmd.Flags().StringVar(&opts.ref, "ref", "", "Credential ref (default: config.yml credential_ref)")
	cmd.Flags().StringVar(&opts.key, "key", "", "Key to set: bot_token or user_token")
	cmd.Flags().BoolVar(&opts.stdin, "stdin", false, "Read the secret value from stdin")
	cmd.Flags().StringVar(&opts.fromEnv, "from-env", "", "Read the secret value from this env var")
	return cmd
}

func run(opts *options) error {
	if opts.key == "" {
		return fmt.Errorf("--key is required (bot_token or user_token)")
	}
	if opts.stdin == (opts.fromEnv != "") {
		return fmt.Errorf("exactly one of --stdin or --from-env is required")
	}

	value, err := readValue(opts)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("empty secret value rejected")
	}

	st, err := keychain.OpenRef(opts.ref)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	switch opts.key {
	case keychain.KeyBotToken:
		err = st.SetBotToken(value)
	case keychain.KeyUserToken:
		err = st.SetUserToken(value)
	default:
		return fmt.Errorf("key %q not allowed (allowed: %s, %s)",
			opts.key, keychain.KeyBotToken, keychain.KeyUserToken)
	}
	if err != nil {
		return err
	}
	// Never echo the value (§1.12); naming the key/ref is fine.
	output.Printf("Stored %s in %s\n", opts.key, st.Ref())
	return nil
}

func readValue(opts *options) (string, error) {
	if opts.fromEnv != "" {
		v := os.Getenv(opts.fromEnv)
		if v == "" {
			// Name the variable (parity with init's resolveBot/resolveUser)
			// so the user knows which env var to populate — never echo a
			// value (§1.12); the var name is not secret.
			return "", fmt.Errorf("--from-env %s is empty or unset", opts.fromEnv)
		}
		return v, nil
	}
	r := opts.in
	if r == nil {
		r = os.Stdin
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read secret from stdin: %w", err)
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}
