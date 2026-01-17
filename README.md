# slack-chat-api

A lightweight command-line interface for interacting with the Slack Web API.

## Not the Official Slack CLI

This project is **not** affiliated with Slack or Salesforce. If you're looking to build Slack apps with workflows, triggers, and datastores, check out the [official Slack CLI](https://api.slack.com/automation/cli).

**What's the difference?**

| Feature | slack-chat-api (this project) | Official Slack CLI |
|---------|-------------------------------|-------------------|
| **Purpose** | Direct API access for automation & scripting | Build and deploy Slack apps |
| **Use cases** | Send messages, manage channels, search, CI/CD integration | Workflows, triggers, datastores, app development |
| **Authentication** | Bot/User OAuth tokens | Slack app credentials |
| **Complexity** | Simple, single binary | Full development framework |

**When to use slack-chat-api:**
- Sending notifications from CI/CD pipelines
- Automating channel management
- Scripting message operations
- Quick API interactions from the terminal
- Integrating Slack into shell scripts or automation tools

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap open-cli-collective/tap
brew install --cask slack-chat-api
```

### From Source

```bash
go install github.com/open-cli-collective/slack-chat-api@latest
```

### Manual Build

```bash
git clone https://github.com/open-cli-collective/slack-chat-api.git
cd slack-chat-api
make build
```

## Platform Support

| Platform | Credential Storage |
|----------|-------------------|
| macOS | Secure (Keychain) |
| Linux | Config file (`~/.config/slack-chat-api/credentials`) |

**Note:** On Linux, credentials are stored in a file with restricted permissions (0600). While not as secure as macOS Keychain, this is standard practice for CLI tools on Linux.

## Authentication

### Quick Setup (2 minutes)

1. Go to [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From an app manifest**
2. Select your workspace
3. Paste this manifest (YAML tab):
   ```yaml
   display_information:
     name: Slack Chat API
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
   slack-chat-api config set-token
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
slack-chat() {
  if [[ -z "$SLACK_API_TOKEN" ]]; then
    export SLACK_API_TOKEN="$(op read 'op://Personal/slack-chat-api/api_token')"
  fi
  command slack-chat-api "$@"
}
```

Or create an alias that always fetches fresh:

```bash
alias slack-chat='SLACK_API_TOKEN="$(op read '\''op://Personal/slack-chat-api/api_token'\'')" slack-chat-api'
```

Replace `op://Personal/slack-chat-api/api_token` with your 1Password secret reference.

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
slack-chat-api config set-token xoxb-your-bot-token

# Set user token (for search)
slack-chat-api config set-token xoxp-your-user-token
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
slack-chat-api channels list

# List with options
slack-chat-api channels list --types public_channel,private_channel  # Include private channels
slack-chat-api channels list --limit 50                              # Limit results
slack-chat-api channels list --exclude-archived=false                # Include archived channels

# Get channel info
slack-chat-api channels get C1234567890

# Create a channel
slack-chat-api channels create my-new-channel
slack-chat-api channels create private-channel --private

# Archive/unarchive
slack-chat-api channels archive C1234567890
slack-chat-api channels unarchive C1234567890

# Set topic/purpose
slack-chat-api channels set-topic C1234567890 "New topic"
slack-chat-api channels set-purpose C1234567890 "Channel purpose"

# Invite users
slack-chat-api channels invite C1234567890 U1111111111 U2222222222
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
slack-chat-api users list
slack-chat-api users list --limit 50

# Get user info
slack-chat-api users get U1234567890

# Search users
slack-chat-api users search "john"
slack-chat-api users search "john@company.com" --field email
slack-chat-api users search "John Smith" --field display_name
slack-chat-api users search "bot" --include-bots
```

#### Users Command Reference

| Command | Flags | Description |
|---------|-------|-------------|
| `list` | `--limit` | List all users |
| `get <id>` | | Get user details |
| `search <query>` | `--limit`, `--field`, `--include-bots` | Search users by name, email, or display name |

#### Users Search Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | `1000` | Maximum users to search through |
| `--field` | `all` | Search field: `all`, `name`, `email`, `display_name` |
| `--include-bots` | `false` | Include bot users in results |

### Messages

```bash
# Send a message (uses Block Kit formatting by default)
slack-chat-api messages send C1234567890 "Hello, *world*!"

# Send from stdin (use "-" as text argument)
echo "Hello from stdin" | slack-chat-api messages send C1234567890 -
cat message.txt | slack-chat-api messages send C1234567890 -

# Send plain text (no formatting)
slack-chat-api messages send C1234567890 "Plain text" --simple

# Send with custom Block Kit blocks
slack-chat-api messages send C1234567890 "Fallback" --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Bold*"}}]'

# Reply in a thread
slack-chat-api messages send C1234567890 "Thread reply" --thread 1234567890.123456

# Update a message
slack-chat-api messages update C1234567890 1234567890.123456 "Updated text"
slack-chat-api messages update C1234567890 1234567890.123456 "Plain update" --simple

# Delete a message
slack-chat-api messages delete C1234567890 1234567890.123456

# Get channel history
slack-chat-api messages history C1234567890
slack-chat-api messages history C1234567890 --limit 50
slack-chat-api messages history C1234567890 --oldest 1234567890.000000  # After this time
slack-chat-api messages history C1234567890 --latest 1234567890.000000  # Before this time

# Get thread replies
slack-chat-api messages thread C1234567890 1234567890.123456
slack-chat-api messages thread C1234567890 1234567890.123456 --limit 50

# Add/remove reactions
slack-chat-api messages react C1234567890 1234567890.123456 thumbsup
slack-chat-api messages unreact C1234567890 1234567890.123456 thumbsup
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
slack-chat-api search messages "quarterly report"
slack-chat-api search messages "in:#general bug fix"
slack-chat-api search messages "from:@alice project update"

# Search files
slack-chat-api search files "budget spreadsheet"
slack-chat-api search files "type:pdf report"

# Search all (messages + files)
slack-chat-api search all "project proposal"
slack-chat-api search all "quarterly" --sort timestamp

# With pagination
slack-chat-api search messages "error" --count 50 --page 2

# Using query builder flags (alternative to modifiers in query string)
slack-chat-api search messages "meeting" --in "#general"
slack-chat-api search messages "update" --from "@alice"
slack-chat-api search messages "report" --scope public
slack-chat-api search messages "project" --after 2025-01-01 --before 2025-12-31
slack-chat-api search messages "link" --has-link
slack-chat-api search files "document" --type pdf
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
| `messages <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight`, `--scope`, `--in`, `--from`, `--after`, `--before`, `--has-link`, `--has-reaction` | Search messages |
| `files <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight`, `--scope`, `--in`, `--from`, `--after`, `--before`, `--type`, `--has-pin` | Search files |
| `all <query>` | `--count`, `--page`, `--sort`, `--sort-dir`, `--highlight`, `--scope`, `--in`, `--from`, `--after`, `--before`, `--has-link`, `--has-reaction` | Search messages and files |

#### Search Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--count` | `-c` | `20` | Results per page (max 100) |
| `--page` | `-p` | `1` | Page number (max 100) |
| `--sort` | `-s` | `score` | Sort by: `score` or `timestamp` |
| `--sort-dir` | | `desc` | Sort direction: `asc` or `desc` |
| `--highlight` | | `false` | Highlight matching terms |

#### Query Builder Flags

These flags provide an alternative to using modifiers in the query string:

| Flag | Description |
|------|-------------|
| `--scope` | Search scope: `all`, `public`, `private`, `dm`, `mpim` |
| `--in` | Filter by channel (e.g., `#general` or `general`) |
| `--from` | Filter by user (e.g., `@alice` or `alice`) |
| `--after` | Content after date (YYYY-MM-DD) |
| `--before` | Content before date (YYYY-MM-DD) |
| `--has-link` | Messages containing links |
| `--has-reaction` | Messages with reactions |
| `--type` | File type filter (files only, e.g., `pdf`, `image`) |
| `--has-pin` | Files that are pinned (files only) |

### Workspace

```bash
# Get workspace info
slack-chat-api workspace info
```

### Config

```bash
# Set API token (interactive prompt)
slack-chat-api config set-token

# Set API token directly
slack-chat-api config set-token xoxb-your-token-here

# Show current config status
slack-chat-api config show

# Delete stored token
slack-chat-api config delete-token
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
slack-chat-api channels list

# JSON output (for scripting)
slack-chat-api channels list --output json
slack-chat-api users get U1234567890 -o json

# Table output (aligned columns)
slack-chat-api channels list --output table
```

### Shell Completion

```bash
# Bash
slack-chat-api completion bash > /etc/bash_completion.d/slack-chat-api

# Zsh
slack-chat-api completion zsh > "${fpath[1]}/_slack-chat-api"

# Fish
slack-chat-api completion fish > ~/.config/fish/completions/slack-chat-api.fish

# PowerShell
slack-chat-api completion powershell > slack-chat-api.ps1
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
