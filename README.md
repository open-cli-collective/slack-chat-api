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

## Usage

### Channels

```bash
# List all channels
slack-cli channels list

# List private channels too
slack-cli channels list --types public_channel,private_channel

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

### Users

```bash
# List all users
slack-cli users list

# Get user info
slack-cli users get U1234567890
```

### Messages

```bash
# Send a message
slack-cli messages send C1234567890 "Hello, world!"

# Reply in a thread
slack-cli messages send C1234567890 "Thread reply" --thread 1234567890.123456

# Update a message
slack-cli messages update C1234567890 1234567890.123456 "Updated text"

# Delete a message
slack-cli messages delete C1234567890 1234567890.123456

# Get channel history
slack-cli messages history C1234567890
slack-cli messages history C1234567890 --limit 50

# Get thread replies
slack-cli messages thread C1234567890 1234567890.123456

# Add/remove reactions
slack-cli messages react C1234567890 1234567890.123456 thumbsup
slack-cli messages unreact C1234567890 1234567890.123456 thumbsup
```

### Workspace

```bash
# Get workspace info
slack-cli workspace info
```

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

### Global Flags

```bash
# Disable colored output
slack-cli channels list --no-color

# Output format (text, json, table)
slack-cli users list --output json
```

### Shell Completion

```bash
# Bash
slack-cli completion bash > /etc/bash_completion.d/slack-cli

# Zsh
slack-cli completion zsh > "${fpath[1]}/_slack-cli"

# Fish
slack-cli completion fish > ~/.config/fish/completions/slack-cli.fish
```

## Aliases

Commands have convenient aliases:

- `channels` → `ch`
- `users` → `u`
- `messages` → `msg` or `m`
- `workspace` → `ws` or `team`

## License

MIT
