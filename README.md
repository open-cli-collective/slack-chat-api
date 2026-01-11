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
         - channels:read
         - channels:write
         - chat:write
         - groups:read
         - im:read
         - mpim:read
         - reactions:write
         - team:read
         - users:read
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

Your token is stored securely in macOS Keychain.

### Alternative: Environment Variable

```bash
export SLACK_API_TOKEN=xoxb-your-token-here
```

### Required Scopes

The manifest above includes these scopes:

- `channels:read` - List channels
- `channels:write` - Create/archive channels
- `chat:write` - Send messages
- `users:read` - List users
- `reactions:write` - Add/remove reactions
- `team:read` - Get workspace info

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
| `archive <id>` | | Archive a channel |
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
| `send <channel> <text>` | `--thread`, `--blocks`, `--simple` | Send a message |
| `update <channel> <ts> <text>` | `--blocks`, `--simple` | Update a message |
| `delete <channel> <ts>` | | Delete a message |
| `history <channel>` | `--limit`, `--oldest`, `--latest` | Get channel history |
| `thread <channel> <ts>` | `--limit` | Get thread replies |
| `react <channel> <ts> <emoji>` | | Add reaction |
| `unreact <channel> <ts> <emoji>` | | Remove reaction |

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

| Command | Description |
|---------|-------------|
| `set-token [token]` | Set API token (prompts if not provided) |
| `show` | Show current configuration status |
| `delete-token` | Delete stored API token |

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
| `workspace` | `ws`, `team` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `SLACK_API_TOKEN` | API token (overrides stored token) |
| `NO_COLOR` | Disable colored output when set |
| `XDG_CONFIG_HOME` | Custom config directory (default: `~/.config`) |

## License

MIT
