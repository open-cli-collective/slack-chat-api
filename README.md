# slack-cli

A command-line interface for Slack.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap piekstra/tap
brew install --cask slack-cli
```

### From Source

```bash
go install github.com/piekstra/slack-cli@latest
```

### Manual Build

```bash
git clone https://github.com/piekstra/slack-cli.git
cd slack-cli
make build
```

## Platform Support

| Platform | Credential Storage |
|----------|-------------------|
| macOS | Secure (Keychain) |
| Linux | Config file (`~/.config/slack-cli/credentials`) |

**Note:** On Linux, credentials are stored in a file with restricted permissions (0600). While not as secure as macOS Keychain, this is standard practice for CLI tools on Linux.

## Authentication

### Quick Setup (2 minutes)

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From an app manifest**
2. Select your workspace
3. Paste this manifest (YAML tab):
   ```yaml
   display_information:
     name: Slack CLI
   oauth_config:
     scopes:
       bot:
         - channels:history
         - channels:manage
         - channels:read
         - chat:write
         - groups:history
         - groups:read
         - reactions:write
         - team:read
         - users:read
       user:
         - search:read
   settings:
     org_deploy_enabled: false
     socket_mode_enabled: false
   ```
4. Click **Create** → **Install to Workspace** → **Allow**
5. Copy the **Bot User OAuth Token** (starts with `xoxb-`)
6. Run:
   ```bash
   slack-cli config set-token
   # Paste your token when prompted
   ```

Your token is stored securely in macOS Keychain (or config file on Linux).

### Alternative: Environment Variable

```bash
export SLACK_API_TOKEN=xoxb-your-token-here
```

### Alternative: 1Password Integration

Use a shell function to lazy-load your token from 1Password on first use:

```bash
# Add to ~/.zshrc or ~/.bashrc
slack() {
  if [[ -z "$SLACK_API_TOKEN" ]]; then
    export SLACK_API_TOKEN="$(op read 'op://Personal/slack-cli/api_token')"
  fi
  command slack-cli "$@"
}
```

Or create an alias that always fetches fresh:

```bash
alias slack='SLACK_API_TOKEN="$(op read '\''op://Personal/slack-cli/api_token'\'')" slack-cli'
```

Replace `op://Personal/slack-cli/api_token` with your 1Password secret reference.

### Required Scopes

The manifest above includes these scopes:

| Scope | Purpose |
|-------|---------|
| `channels:read` | List public channels, get channel info |
| `channels:history` | Read message history from public channels |
| `channels:manage` | Create, archive, set topic/purpose, invite users |
| `chat:write` | Send, update, delete messages |
| `groups:read` | List private channels |
| `groups:history` | Read message history from private channels |
| `reactions:write` | Add/remove reactions |
| `team:read` | Get workspace info |
| `users:read` | List users, get user info |
| `search:read` | Search messages and files (user token only) |

### Token Types

This CLI supports two types of Slack tokens:

| Token Type | Prefix | Commands | How to Get |
|------------|--------|----------|------------|
| Bot token | `xoxb-` | channels, users, messages, workspace | OAuth & Permissions → Bot User OAuth Token |
| User token | `xoxp-` | search | OAuth & Permissions → User OAuth Token |

Most commands use the **bot token**. Search commands require a **user token**.

**Setting up both tokens:**

```bash
# Set bot token (for channels, users, messages, workspace)
slack-cli config set-token xoxb-your-bot-token

# Set user token (for search)
slack-cli config set-token xoxp-your-user-token
```

The `set-token` command automatically detects the token type and stores it appropriately.

**Getting a user token:**

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → Your app
2. OAuth & Permissions → User Token Scopes → Add `search:read`
3. Reinstall app to workspace (if already installed)
4. Copy the **User OAuth Token** (starts with `xoxp-`)

**Environment variables:**

| Variable | Token Type | Description |
|----------|------------|-------------|
| `SLACK_API_TOKEN` | Bot | Bot token for most commands |
| `SLACK_USER_TOKEN` | User | User token for search commands |

## Global Flags

These flags are available on all commands:

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `text` | Output format: `text`, `json`, or `table` |
| `--no-color` | | `false` | Disable colored output |
| `--version` | `-v` | | Show version information |
| `--help` | `-h` | | Show help for any command |

## Usage

### Channels

```bash
# List all channels
slack-cli channels list

# List with options
slack-cli channels list --types public_channel,private_channel  # Include private channels
slack-cli channels list --limit 50                              # Limit results
slack-cli channels list --exclude-archived=false                # Include archived channels

# Get channel info
slack-cli channels get C1234567890

# Create a channel
slack-cli channels create my-new-channel
slack-cli channels create private-channel --private

# Archive/unarchive
slack-cli channels archive C1234567890
slack-cli channels unarchive C1234567890

# Set topic/purpose
slack-cli channels set-topic C1234567890 "New topic"
slack-cli channels set-purpose C1234567890 "Channel purpose"

# Invite users
slack-cli channels invite C1234567890 U1111111111 U2222222222
```

#### Channels Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `list` | `--types`, `--limit`, `--exclude-archived` | List channels |
| `get <id>` | | Get channel details |
| `create <name>` | `--private` | Create a channel |
| `archive <id>` | `--force` | Archive a channel (prompts for confirmation) |
| `unarchive <id>` | | Unarchive a channel |
| `set-topic <id> <topic>` | | Set channel topic |
| `set-purpose <id> <purpose>` | | Set channel purpose |
| `invite <id> <user>...` | | Invite users to channel |

### Users

```bash
# List all users
slack-cli users list
slack-cli users list --limit 50

# Get user info
slack-cli users get U1234567890
```

#### Users Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `list` | `--limit` | List all users |
| `get <id>` | | Get user details |

### Messages

```bash
# Send a message (uses Block Kit formatting by default)
slack-cli messages send C1234567890 "Hello, *world*!"

# Send from stdin (use "-" as text argument)
echo "Hello from stdin" | slack-cli messages send C1234567890 -
cat message.txt | slack-cli messages send C1234567890 -

# Send plain text (no formatting)
slack-cli messages send C1234567890 "Plain text" --simple

# Send with custom Block Kit blocks
slack-cli messages send C1234567890 "Fallback" --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Bold*"}}]'

# Reply in a thread
slack-cli messages send C1234567890 "Thread reply" --thread 1234567890.123456

# Update a message
slack-cli messages update C1234567890 1234567890.123456 "Updated text"
slack-cli messages update C1234567890 1234567890.123456 "Plain update" --simple

# Delete a message
slack-cli messages delete C1234567890 1234567890.123456

# Get channel history
slack-cli messages history C1234567890
slack-cli messages history C1234567890 --limit 50
slack-cli messages history C1234567890 --oldest 1234567890.000000  # After this time
slack-cli messages history C1234567890 --latest 1234567890.000000  # Before this time

# Get thread replies
slack-cli messages thread C1234567890 1234567890.123456
slack-cli messages thread C1234567890 1234567890.123456 --limit 50

# Add/remove reactions
slack-cli messages react C1234567890 1234567890.123456 thumbsup
slack-cli messages unreact C1234567890 1234567890.123456 thumbsup
```

#### Messages Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `send <channel> <text>` | `--thread`, `--blocks`, `--simple` | Send a message (use `-` for stdin) |
| `update <channel> <ts> <text>` | `--blocks`, `--simple` | Update a message |
| `delete <channel> <ts>` | `--force` | Delete a message (prompts for confirmation) |
| `history <channel>` | `--limit`, `--oldest`, `--latest` | Get channel history |
| `thread <channel> <ts>` | `--limit` | Get thread replies |
| `react <channel> <ts> <emoji>` | | Add reaction |
| `unreact <channel> <ts> <emoji>` | | Remove reaction |

### Search

> **Note:** Search requires a user token (`xoxp-*`). See [Token Types](#token-types).

```bash
# Search messages
slack-cli search messages "quarterly report"
slack-cli search messages "in:#general bug fix"
slack-cli search messages "from:@alice project update"

# Search files
slack-cli search files "budget spreadsheet"
slack-cli search files "type:pdf report"

# Search all (messages + files)
slack-cli search all "project proposal"
slack-cli search all "quarterly" --sort timestamp

# With pagination
slack-cli search messages "error" --count 50 --page 2
```

#### Search Modifiers

| Modifier | Example | Description |
|----------|---------|-------------|
| `in:` | `in:#channel` or `in:@user` | Search in specific channel or DM |
| `from:` | `from:@username` | Content from specific user |
| `before:` | `before:2025-01-01` | Before date |
| `after:` | `after:2025-01-01` | After date |
| `has:link` | | Messages containing links |
| `has:reaction` | | Messages with reactions |
| `type:` | `type:pdf` | Files of specific type |

#### Search Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `messages <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight` | Search messages |
| `files <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight` | Search files |
| `all <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight` | Search messages and files |

#### Search Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--count` | `-c` | `20` | Results per page (max 100) |
| `--page` | `-p` | `1` | Page number (max 100) |
| `--sort` | `-s` | `score` | Sort by: `score` or `timestamp` |
| `--sort-dir` | | `desc` | Sort direction: `asc` or `desc` |
| `--highlight` | | `false` | Highlight matching terms |

### Workspace

```bash
# Get workspace info
slack-cli workspace info
```

### Config

```bash
# Set API token (interactive prompt)
slack-cli config set-token

# Set API token directly
slack-cli config set-token xoxb-your-token-here

# Show current config status
slack-cli config show

# Delete stored token
slack-cli config delete-token
```

#### Config Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `set-token [token]` | | Set API token (auto-detects bot/user type) |
| `show` | | Show current configuration status |
| `delete-token` | `--force`, `--type` | Delete stored token(s) |
| `test` | | Test authentication for configured tokens |

The `delete-token` command accepts a `--type` flag:
- `--type bot` - Delete only the bot token
- `--type user` - Delete only the user token
- `--type all` - Delete both tokens (default)

### Output Formats

All commands support multiple output formats via the `--output` (or `-o`) flag:

```bash
# Default text output
slack-cli channels list

# JSON output (for scripting)
slack-cli channels list --output json
slack-cli users get U1234567890 -o json

# Table output (aligned columns)
slack-cli channels list --output table
```

### Shell Completion

```bash
# Bash
slack-cli completion bash > /etc/bash_completion.d/slack-cli

# Zsh
slack-cli completion zsh > "${fpath[1]}/_slack-cli"

# Fish
slack-cli completion fish > ~/.config/fish/completions/slack-cli.fish

# PowerShell
slack-cli completion powershell > slack-cli.ps1
```

## Aliases

Commands have convenient aliases:

| Command | Aliases |
|---------|---------|
| `channels` | `ch` |
| `users` | `u` |
| `messages` | `msg`, `m` |
| `search` | `s` |
| `workspace` | `ws`, `team` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SLACK_API_TOKEN` | Bot token (overrides stored bot token) |
| `SLACK_USER_TOKEN` | User token for search (overrides stored user token) |
| `NO_COLOR` | Disable colored output when set |
| `XDG_CONFIG_HOME` | Custom config directory (default: `~/.config`) |

## Known Limitations

### Unarchiving Channels

Bot tokens (`xoxb-`) cannot unarchive channels due to a [Slack API limitation](https://api.slack.com/methods/conversations.unarchive). When a channel is archived, the bot is automatically removed from it, and bot tokens require membership to unarchive.

**Workarounds:**
- Unarchive channels via the Slack UI
- Use a user token (`xoxp-`) instead of a bot token

## License

MIT
