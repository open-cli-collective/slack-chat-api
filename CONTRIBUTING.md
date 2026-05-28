# Contributing to slack-chat-api

Thank you for your interest in contributing to slack-chat-api!

## Development Setup

### Prerequisites

- Go 1.24 or later
- Make
- golangci-lint v2.0+ (for linting)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/open-cli-collective/slack-chat-api.git
cd slack-chat-api

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

## Development Workflow

1. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the code patterns below

3. **Test your changes**:
   ```bash
   make test    # Run all tests
   make lint    # Check for lint issues
   make build   # Ensure it compiles
   ```

4. **Commit with a descriptive message**

5. **Push and create a Pull Request**

## Code Patterns

### Command Structure

Commands live in `internal/cmd/<resource>/` directories. Each command should:

1. Define an options struct for flags
2. Use injectable client for testability
3. Render via `output.Table()` / `output.KeyValue()` / `output.Printf()` — text and table only. Per #173 (cli-common `docs/output-and-rendering.md` §2), do NOT add `--json` flags or `output.IsJSON()` branches to resource or mutation-success commands; JSON is reserved for control-plane carve-outs (see "Control-plane carve-outs" below).

Example:

```go
type myOptions struct {
    someFlag string
}

func newMyCmd() *cobra.Command {
    opts := &myOptions{}
    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "Does something",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runMy(opts, nil)
        },
    }
    cmd.Flags().StringVar(&opts.someFlag, "flag", "", "Description")
    return cmd
}

func runMy(opts *myOptions, c *client.Client) error {
    if c == nil {
        var err error
        c, err = client.New()
        if err != nil {
            return err
        }
    }
    // Implementation...
}
```

### Output Formatting

Resource and mutation-success commands emit text or table only. Per #173, JSON is reserved for local control-plane carve-outs (today only `slck config show --json`). Examples:

```go
// Detail view
output.Printf("Result: %s\n", result.Name)
```

For list commands, use `output.Table()`:

```go
headers := []string{"ID", "Name"}
rows := make([][]string, len(items))
for i, item := range items {
    rows[i] = []string{item.ID, item.Name}
}
output.Table(headers, rows)
```

### Control-plane carve-outs

A new command qualifies as a control-plane carve-out (and may declare a local `--json` boolean flag) only if it (a) lives outside the resource-command packages and (b) emits a write confirmation envelope (`set-credential`-style) or diagnostic introspection of CLI state (`config show`-style) — not a Slack API resource. Carve-outs call `output.PrintJSON(envelope)` directly when `--json` is set; the global `-o text/table` is ignored under the local `--json`.

### Testing

Tests should use mock clients:

```go
func TestRunMy(t *testing.T) {
    // Setup mock client
    mockClient := setupMockClient(t)

    opts := &myOptions{someFlag: "value"}
    err := runMy(opts, mockClient)

    assert.NoError(t, err)
}
```

## Pull Request Guidelines

- Reference any related GitHub issues (e.g., "Fixes #123")
- Keep PRs focused on a single change
- Ensure all tests pass (`make test`)
- Ensure lint passes (`make lint`)
- Update documentation if adding new features

## Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports` (enforced by linter)
- Keep functions focused and testable
- Add comments for non-obvious logic

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new feature
fix: fix a bug
docs: update documentation
test: add tests
refactor: refactor code
ci: update CI configuration
chore: maintenance tasks
```

Examples:
```
feat: add channel archive command
fix: handle rate limiting in message send
docs: update installation instructions
```

## Project Structure

```
slack-chat-api/
├── cmd/slck/             # Entry point
├── internal/
│   ├── cmd/              # Command implementations
│   │   ├── root/         # Root command and global flags
│   │   ├── channels/     # Channel commands
│   │   ├── users/        # User commands
│   │   ├── messages/     # Message commands
│   │   ├── workspace/    # Workspace info command
│   │   └── config/       # Token management commands
│   ├── client/           # Slack API client wrapper
│   ├── keychain/         # Secure credential storage
│   ├── output/           # Output formatting (text/json/table)
│   └── version/          # Build-time version injection
└── .github/              # GitHub workflows and templates
```

## Reporting Issues

When reporting bugs, please include:

- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Any relevant error messages

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
