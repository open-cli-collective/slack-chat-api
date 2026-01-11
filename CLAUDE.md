# CLAUDE.md

This file provides context for AI assistants working with this codebase.

## Project Overview

A command-line interface for Slack, supporting channel management, user lookup, messaging, and workspace info.

## Quick Commands

```bash
make build      # Build binary to ./bin/slack-cli
make test       # Run tests with race detection and coverage
make lint       # Run golangci-lint
make clean      # Remove build artifacts
make install    # Install to $GOPATH/bin
```

## Project Structure

```
slack-cli/
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
- `--workspace, -w` - Workspace override (not yet implemented)

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

- **Token not found**: Run `slack-cli config set-token` or set `SLACK_API_TOKEN`
- **Permission denied**: Check bot token scopes in Slack app settings
- **Lint failures**: Run `make lint` locally before pushing
- **golangci-lint version**: CI uses v2.0.2 with v2 config format

## Dependencies

- `github.com/slack-go/slack` - Slack API client
- `github.com/spf13/cobra` - CLI framework
- `github.com/zalando/go-keyring` - Cross-platform keychain
