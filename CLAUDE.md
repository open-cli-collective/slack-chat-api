# CLAUDE.md

This file provides context for AI assistants working with this codebase.

## Project Overview

A command-line interface for Slack, supporting channel management, user lookup, messaging, and workspace info.

## Quick Commands

```bash
make build      # Build binary to ./bin/slck
make test       # Run tests with race detection and coverage
make lint       # Run golangci-lint
make clean      # Remove build artifacts
make install    # Install to $GOPATH/bin
```

## Project Structure

```
slack-chat-api/
├── main.go                     # Entry point
├── internal/
│   ├── cmd/                    # Command implementations
│   │   ├── root/               # Root command and global flags
│   │   ├── channels/           # Channel commands (list, get, create, etc.)
│   │   ├── users/              # User commands (list, get)
│   │   ├── messages/           # Message commands (send, history, react, etc.)
│   │   ├── workspace/          # Workspace info command
│   │   └── config/             # Token management commands
│   ├── client/                 # Slack API client wrapper
│   ├── keychain/               # Secure credential storage
│   ├── output/                 # Output formatting (text/json/table)
│   └── version/                # Build-time version injection
├── .github/workflows/ci.yml    # CI pipeline
├── .golangci.yml               # Linter configuration (v2 format)
└── Makefile                    # Build targets
```

## Key Patterns

### Options Struct Pattern

All commands use an options struct with an injectable client for testability:

```go
type listOptions struct {
    types           string
    excludeArchived bool
    limit           int
}

func runList(opts *listOptions, c *client.Client) error {
    if c == nil {
        var err error
        c, err = client.New()
        if err != nil {
            return err
        }
    }
    // Business logic...
}
```

### Output Formatting

Commands support `--output text|json|table` via the `internal/output` package:

```go
if output.IsJSON() {
    return output.PrintJSON(data)
}
output.Table(headers, rows)  // For list commands
output.KeyValue("ID", item.ID)  // For detail views
```

### Global Flags

- `--output, -o` - Output format: text (default), json, or table
- `--no-color` - Disable colored output

## Testing

Tests use mock clients injected via the options struct:

```go
func TestRunList(t *testing.T) {
    mockClient := &client.Client{...}  // Mock setup
    opts := &listOptions{limit: 10}
    err := runList(opts, mockClient)
    // Assertions...
}
```

Run tests: `make test`

Coverage report: `go tool cover -html=coverage.out`

## API Client

The `internal/client` package wraps the Slack API:

- `client.New()` - Creates client from token (env var or keychain)
- All API calls return typed responses
- Pagination handled internally with configurable limits

## Adding a New Command

1. Create file in appropriate `internal/cmd/<resource>/` directory
2. Define options struct with flags
3. Implement `newXxxCmd()` returning `*cobra.Command`
4. Implement `runXxx(opts, client)` with business logic
5. Register in the resource's root command
6. Add tests using mock client injection

## Common Issues

- **Token not found**: Run `slck init` or `slck set-credential --key bot_token --stdin`. Environment variables are NOT read at runtime (only as setup ingress, e.g. `init --bot-token-from-env`).
- **Permission denied**: Check bot token scopes in Slack app settings
- **Lint failures**: Run `make lint` locally before pushing
- **golangci-lint version**: CI uses v2.0.2 with v2 config format

## Credentials

slck stores credentials in the OS keyring via `cli-common/credstore` (Open
CLI Collective Secret-Handling Standard §2.4). The `internal/keychain`
package is a credstore adapter (no `security` shell-out, no plaintext file).
Non-secret config (`credential_ref`, `workspace`, `keyring.backend`) lives in
`~/.config/slack-chat-api/config.yml`. Ingress is only `slck init` /
`slck set-credential` (stdin or `--from-env`); never a flag/positional value.

## Dependencies

- `github.com/open-cli-collective/cli-common` - shared credstore (OS keyring)
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/term` - no-echo passphrase prompt (file-backend opt-in)

(The HTTP Slack client is hand-rolled in `internal/client`; there is no
`slack-go`/`zalando` dependency.)

## Commit Conventions

Use conventional commits:

```
type(scope): description

feat(channels): add archive command
fix(messages): handle rate limiting
docs(readme): add configuration examples
```

| Prefix | Purpose | Triggers Release? |
|--------|---------|-------------------|
| `feat:` | New features | Yes |
| `fix:` | Bug fixes | Yes |
| `docs:` | Documentation only | No |
| `test:` | Adding/updating tests | No |
| `refactor:` | Code changes that don't fix bugs or add features | No |
| `chore:` | Maintenance tasks | No |
| `ci:` | CI/CD changes | No |

## CI & Release Workflow

Releases are automated with a dual-gate system to avoid unnecessary releases:

**Gate 1 - Path filter:** Only triggers when Go code changes (`**.go`, `go.mod`, `go.sum`)
**Gate 2 - Commit prefix:** Only `feat:` and `fix:` commits create releases

This means:
- `feat: add command` + Go files changed → release
- `fix: handle edge case` + Go files changed → release
- `docs:`, `ci:`, `test:`, `refactor:` → no release
- Changes only to docs, packaging, workflows → no release

**After merging a release-triggering PR:** The workflow creates a tag, which triggers GoReleaser to build binaries and publish to Homebrew. Chocolatey and Winget require manual workflow dispatch.
