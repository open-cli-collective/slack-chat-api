# Integration Tests

Manual integration tests for verifying slck against a live Slack workspace. Tests are organized from safe (read-only) to destructive, so you can stop at any section.

> **Credential model (§1.11/§1.12):** slck stores credentials in the OS
> keyring only. Ingress is `slck init` or `slck set-credential` (stdin /
> `--from-env`) — never a positional/flag value. `slck config set-token` is
> removed. `SLACK_API_TOKEN` / `SLACK_USER_TOKEN` are **not** read at
> runtime (only as `init --bot-token-from-env` setup ingress). Scenarios
> below use the new commands.

---

## Part 1: Setup

### Step 1: Create Slack App

1. Go to https://api.slack.com/apps
2. Click "Create New App" → "From scratch"
3. Name it (e.g., "slack-chat-api-test") and select your workspace

### Step 2: Configure Token Scopes

In **OAuth & Permissions**, add these **Bot Token Scopes**:

| Scope | Purpose | Required For |
|-------|---------|--------------|
| `channels:read` | List public channels, get channel info | Part 2 |
| `channels:history` | Read message history | Part 2 |
| `groups:read` | List private channels | Part 2 |
| `groups:history` | Read private channel history | Part 2 |
| `users:read` | List users, get user info | Part 2 |
| `users:read.email` | See user email addresses | Part 2 |
| `team:read` | Get workspace info | Part 2 |
| `chat:write` | Send, update, delete messages | Part 3 |
| `reactions:write` | Add/remove reactions | Part 3 |
| `channels:manage` | Create, archive, set topic/purpose, invite | Parts 4 & 5 |
| `groups:write` | Topic/purpose/invite for private channels | Part 4 |
| `emoji:read` | List custom workspace emoji | Part 7 |
| `files:read` | Download files, get file info | Part 7 |

**Note:** `channels:manage` is a superset that includes `channels:write.topic` and `channels:write.invites`. You can use the granular scopes instead if you want more limited permissions.

Also add this **User Token Scope** (for Part 3B: Search Tests):

| Scope | Purpose | Required For |
|-------|---------|--------------|
| `search:read` | Search messages and files | Part 3B |

### Step 3: Install App & Configure CLI

```bash
# Install app to workspace (in Slack app settings)

# Build the CLI
make build

# Set up Bot + User tokens (guided; input is not echoed)
# Copy the "Bot User OAuth Token" (xoxb-) and "User OAuth Token" (xoxp-)
./bin/slck init
# Or non-interactively, per secret, from a pipe:
#   printf '%s' "$XOXB" | ./bin/slck set-credential --key bot_token  --stdin
#   printf '%s' "$XOXP" | ./bin/slck set-credential --key user_token --stdin

# Verify bot token works
./bin/slck workspace info

# Verify both tokens are configured
./bin/slck config show
./bin/slck config test
```

### Step 4: Discover Test Inputs

Use the CLI to find the IDs you need (Slack IDs are opaque and not easily visible in the UI):

```bash
# Find your test channel ID (bot must already be in the channel)
# Look for your test channel name in the output
slck channels list

# Example output:
# ID            NAME              MEMBERS
# C08UR9H3YHU   testing           5
# C07ABC123DE   general           42

# Find a user ID (optional, for invite tests)
slck users list --limit 10
```

Set these for easy reference during testing:

```bash
export TEST_CHANNEL_ID="C..."      # From channels list output
export TEST_USER_ID="U..."         # Optional: from users list output
```

| Variable | Description | Required |
|----------|-------------|----------|
| `TEST_CHANNEL_ID` | Channel the bot is already in | Yes |
| `TEST_USER_ID` | A user to invite to channels | Optional (Part 4 only) |

**Prerequisite:** The bot must already be invited to `TEST_CHANNEL_ID`. Use `/invite @your-bot-name` in Slack if needed.

---

## Part 2: Read-Only Tests

**Scopes required:** `team:read`, `users:read`, `channels:read`, `channels:history`

These tests don't modify anything. Safe to run anytime.

### 2.1 Workspace Info

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck workspace info` | Shows workspace ID, name, domain |
| 2 | `slck workspace info -o json` | Valid JSON with `id`, `name`, `domain` |
| 3 | `slck workspace info -o table` | Formatted table output |

### 2.2 Users

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck users list` | Table with ID, USERNAME, REAL NAME |
| 2 | `slck users list --limit 3` | Exactly 3 users |
| 3 | `slck users list -o json` | Valid JSON array |
| 4 | `slck users get $TEST_USER_ID` | User details (ID, name, email, status) |
| 5 | `slck users get $TEST_USER_ID -o json` | Full user object with nested profile |
| 6 | `slck users get UINVALID999` | Error: `user_not_found` |

### 2.3 Channels

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels list` | Table with ID, NAME, MEMBERS |
| 2 | `slck channels list --limit 5` | Exactly 5 channels |
| 3 | `slck channels list --types public_channel` | Only public channels |
| 4 | `slck channels list --exclude-archived=false` | Includes archived channels |
| 5 | `slck channels list -o json` | Valid JSON array |
| 6 | `slck channels get $TEST_CHANNEL_ID` | Channel details (ID, name, topic, purpose, members) |
| 7 | `slck channels get $TEST_CHANNEL_ID -o json` | Full channel object |
| 8 | `slck channels get CINVALID999` | Error: `channel_not_found` |

### 2.4 Message History

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages history $TEST_CHANNEL_ID` | Table with timestamp, user, text |
| 2 | `slck messages history $TEST_CHANNEL_ID --limit 5` | Exactly 5 messages |
| 3 | `slck messages history $TEST_CHANNEL_ID -o json` | Valid JSON array of messages |

### 2.5 Output Formats

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels list -o text` | Same as default (human-readable) |
| 2 | `slck channels list -o json \| jq '.[0].id'` | Works with jq |
| 3 | `slck channels list --no-color` | No ANSI escape codes in output |

---

## Part 3: Messaging Tests

**Scopes required:** `chat:write`, `reactions:write`

These tests create messages, then clean them up at the end.

### 3.1 Send & Verify Message

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `slck messages send $TEST_CHANNEL_ID "Integration test message"` | "Message sent (ts: X)" | **Save TS₁** |
| 2 | `slck messages history $TEST_CHANNEL_ID --limit 1` | Shows your message |
| 3 | `slck messages send $TEST_CHANNEL_ID "JSON test" -o json` | JSON with `ts` field | (verify only) |
| 4 | `slck messages send $TEST_CHANNEL_ID "Plain text" --simple` | Message without Block Kit formatting |
| 5 | `slck messages send --channel $TEST_CHANNEL_ID "Channel flag test"` | "Message sent" (--channel flag alternative) |

### 3.2 Multiline Message (stdin)

| Step | Command | Expected |
|------|---------|----------|
| 1 | `echo -e "Line 1\nLine 2\nLine 3" \| slck messages send $TEST_CHANNEL_ID -` | Multiline message appears in Slack |

### 3.3 Reactions

Using **TS₁** from step 3.1:

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages react $TEST_CHANNEL_ID <TS₁> thumbsup` | "Added :thumbsup: reaction" |
| 2 | `slck messages react $TEST_CHANNEL_ID <TS₁> :heart:` | "Added :heart: reaction" (colons stripped) |
| 3 | `slck messages react $TEST_CHANNEL_ID <TS₁> thumbsup` | "Already reacted with :thumbsup:" (idempotent, exit 0) |
| 4 | `slck messages unreact $TEST_CHANNEL_ID <TS₁> thumbsup` | "Removed :thumbsup: reaction" |
| 5 | `slck messages unreact $TEST_CHANNEL_ID <TS₁> heart` | "Removed :heart: reaction" |

### 3.4 Threading

Using **TS₁** from step 3.1:

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `slck messages send $TEST_CHANNEL_ID "Thread reply" --thread <TS₁>` | "Message sent" as thread reply | **Save TS₂** |
| 2 | `slck messages thread $TEST_CHANNEL_ID <TS₁>` | Shows parent + reply, full text (not truncated) |
| 3 | `slck messages thread $TEST_CHANNEL_ID <TS₁> -o json` | JSON array of thread messages |
| 4 | `slck messages thread $TEST_CHANNEL_ID <TS₁> --since <TS₁>` | Only replies after TS₁ (may exclude parent) |

### 3.5 Update Message

Using **TS₁** from step 3.1:

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages update $TEST_CHANNEL_ID <TS₁> "Updated message text"` | "Message updated" |
| 2 | `slck messages history $TEST_CHANNEL_ID --limit 1` | Shows updated text with `[edited]` suffix |
| 3 | `slck messages thread $TEST_CHANNEL_ID <TS₁>` | Updated message shows `[edited]` suffix |
| 4 | `slck messages thread $TEST_CHANNEL_ID <TS₁> -o json` | Updated message has `"edited"` object with `user` and `ts` |

### 3.6 Cleanup: Delete Messages

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages delete $TEST_CHANNEL_ID <TS₂> --force` | "Message deleted" (thread reply) |
| 2 | `slck messages delete $TEST_CHANNEL_ID <TS₁> --force` | "Message deleted" (parent) |

**Verify:** `slck messages history $TEST_CHANNEL_ID --limit 3` should not show the deleted messages.

---

## Part 3B: Search Tests

**Scopes required:** `search:read` (user token)

Search requires a **user token** (`xoxp-*`), not a bot token. These tests verify search functionality.

### Prerequisites

1. Configure a user token:
   ```bash
   # Get your user token from Slack app settings:
   # OAuth & Permissions → User OAuth Token (starts with xoxp-)
   printf '%s' "$XOXP" | slck set-credential --key user_token --stdin
   # (or run `slck init` for the guided flow)
   ```

2. Verify both tokens are configured:
   ```bash
   slck config show
   # Should show both Bot Token and User Token

   slck config test
   # Should validate both tokens
   ```

### 3B.1 Setup: Create Searchable Content

First, create a message with a unique identifier that we can search for:

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `export SEARCH_ID="integ-test-$(date +%s)"` | Sets unique identifier | **Save SEARCH_ID** |
| 2 | `slck messages send $TEST_CHANNEL_ID "Search test: $SEARCH_ID"` | Message sent | **Save TS₃** |
| 3 | Wait ~5 seconds for Slack to index | (Slack search has indexing delay) | |

### 3B.2 Search Messages

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "$SEARCH_ID"` | Shows the test message with channel, user, timestamp |
| 2 | `slck search messages "$SEARCH_ID" -o json` | Valid JSON with `query`, `messages.matches[]`, `messages.paging` |
| 3 | `slck search messages "$SEARCH_ID" -o table` | Table format output |
| 4 | `slck search messages "in:#$TEST_CHANNEL_NAME $SEARCH_ID"` | Same message (filtered by channel) |

### 3B.3 Search with Pagination

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "test" --count 5` | Max 5 results |
| 2 | `slck search messages "test" --count 5 --page 1` | Page 1 of results |
| 3 | `slck search messages "test" --count 5 --page 2` | Page 2 (may be empty or have different results) |

### 3B.4 Search with Sorting

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "test" --sort score` | Sorted by relevance (default) |
| 2 | `slck search messages "test" --sort timestamp --sort-dir desc` | Most recent first |
| 3 | `slck search messages "test" --sort timestamp --sort-dir asc` | Oldest first |

### 3B.5 Search Files

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search files "document"` | Lists matching files (if any exist) |
| 2 | `slck search files "document" -o json` | Valid JSON with `files.matches[]` |

### 3B.6 Search All

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search all "$SEARCH_ID"` | Shows message result under "Messages" section |
| 2 | `slck search all "$SEARCH_ID" -o json` | JSON with both `messages` and `files` objects |
| 3 | `slck search all "test" --count 10` | Shows both messages and files (if any) |

### 3B.7 Search Error Cases

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "xyznonexistent123"` | "No messages found" |
| 2 | `slck search messages "test" --count 101` | Error: count must be between 1 and 100 |
| 3 | `slck search messages "test" --page 0` | Error: page must be between 1 and 100 |
| 4 | `slck search messages "test" --sort invalid` | Error: sort must be 'score' or 'timestamp' |

### 3B.8 Search Without User Token (Error Case)

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck config delete-token --type user --force` | Delete stored user token |
| 2 | `slck search messages "test"` | Error mentioning user token requirement |
| 3 | `printf '%s' "$XOXP" \| slck set-credential --key user_token --stdin` | Re-configure user token |

### 3B.9 User Search Tests

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck users search "test"` | Lists users matching "test" (or "No users found") |
| 2 | `slck users search "test" -o json` | Valid JSON array |
| 3 | `slck users search "nonexistent12345xyz"` | "No users found" |
| 4 | `slck users search "$TEST_USERNAME" --field name` | Matches by username only |
| 5 | `slck users search "test" --field email` | Matches by email only |
| 6 | `slck users search "bot" --include-bots` | Includes bot users in results |
| 7 | `slck users search "test" --field invalid` | Error: invalid field |

### 3B.10 Search Scope Filter Tests

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "$SEARCH_ID" --scope all` | Same as default (finds test message) |
| 2 | `slck search messages "$SEARCH_ID" --scope public` | Only public channel results |
| 3 | `slck search messages "test" --scope private` | Only private channel results (may be empty) |
| 4 | `slck search messages "test" --scope dm` | Only DM results (may be empty) |
| 5 | `slck search messages "test" --scope invalid` | Error: invalid scope |

### 3B.11 Query Builder Flag Tests

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "test" --in "$TEST_CHANNEL_NAME"` | Results from specific channel |
| 2 | `slck search messages "test" --in "#$TEST_CHANNEL_NAME"` | Same (handles # prefix) |
| 3 | `slck search messages "test" --from "@$TEST_USERNAME"` | Results from specific user |
| 4 | `slck search messages "test" --after "2024-01-01"` | Results after date |
| 5 | `slck search messages "test" --before "2099-12-31"` | Results before date |
| 6 | `slck search messages "test" --after "2024-01-01" --before "2099-12-31"` | Results in date range |
| 7 | `slck search messages "test" --has-link` | Results containing links (may be empty) |
| 8 | `slck search messages "test" --has-reaction` | Results with reactions (may be empty) |
| 9 | `slck search files "test" --type pdf` | Only PDF files (may be empty) |

### 3B.12 Combined Filters Tests

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "$SEARCH_ID" --in "$TEST_CHANNEL_NAME" --scope public` | Combined scope + channel filter |
| 2 | `slck search messages "test" --from "@$TEST_USERNAME" --after "2024-01-01"` | Combined from + date filter |
| 3 | `slck search messages "test" -o json --in "$TEST_CHANNEL_NAME"` | JSON output with filters |
| 4 | `slck search all "test" --scope public --in "$TEST_CHANNEL_NAME"` | Combined filters on search all |

### 3B.13 Query Builder Error Cases

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck search messages "test" --after "invalid-date"` | Error: invalid date format |
| 2 | `slck search messages "test" --before "not-a-date"` | Error: invalid date format |
| 3 | `slck search messages "test" --scope badscope` | Error: invalid scope |

### 3B.14 Cleanup: Delete Search Test Message

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages delete $TEST_CHANNEL_ID <TS₃> --force` | "Message deleted" |

---

## Part 4: Channel Metadata Tests

**Scopes required:** `channels:write`

These tests modify channel metadata but restore original values afterward.

### 4.1 Save Original State

| Step | Command | Capture |
|------|---------|---------|
| 1 | `slck channels get $TEST_CHANNEL_ID -o json` | **Save original TOPIC and PURPOSE** |

### 4.2 Modify Topic & Purpose

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels set-topic $TEST_CHANNEL_ID "Integration test topic"` | "Set topic for channel" |
| 2 | `slck channels get $TEST_CHANNEL_ID` | Topic shows "Integration test topic" |
| 3 | `slck channels set-purpose $TEST_CHANNEL_ID "Integration test purpose"` | "Set purpose for channel" |
| 4 | `slck channels get $TEST_CHANNEL_ID` | Purpose shows "Integration test purpose" |

### 4.3 Invite User (Optional)

Skip if you didn't set `TEST_USER_ID`.

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels invite $TEST_CHANNEL_ID $TEST_USER_ID` | "Invited 1 user(s)" or "User(s) already in channel" (idempotent, exit 0) |

### 4.4 Restore Original State

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels set-topic $TEST_CHANNEL_ID "<original topic>"` | Restored |
| 2 | `slck channels set-purpose $TEST_CHANNEL_ID "<original purpose>"` | Restored |

---

## Part 5: Destructive Tests ⚠️

**Scopes required:** `channels:manage`

**Warning:** These tests create and archive channels. They require elevated permissions and will leave artifacts in your workspace if interrupted.

### 5.1 Create Channels

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `slck channels create test-integ-$(date +%s)` | "Created channel: test-integ-X (C...)" | **Save NEW_CHANNEL_ID** |
| 2 | `slck channels create test-private-$(date +%s) --private` | "Created channel" (private) | **Save PRIVATE_CHANNEL_ID** |
| 3 | `slck channels get <NEW_CHANNEL_ID>` | Shows new channel details |

### 5.2 Channel Creation Errors

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels create general` | Error: `name_taken` |
| 2 | `slck channels create "has spaces"` | Error: `invalid_name_specials` |

### 5.3 Archive & Unarchive

Using **NEW_CHANNEL_ID** from step 5.1:

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels archive <NEW_CHANNEL_ID> --force` | "Archived channel" |
| 2 | `slck channels archive <NEW_CHANNEL_ID> --force` | "Channel already archived" (idempotent, exit 0) |
| 3 | `slck channels unarchive <NEW_CHANNEL_ID>` | "Unarchived channel" |
| 4 | `slck channels unarchive <NEW_CHANNEL_ID>` | "Channel not archived" (idempotent, exit 0) |

### 5.4 Cleanup: Archive Test Channels

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels archive <NEW_CHANNEL_ID> --force` | "Archived channel" |
| 2 | `slck channels archive <PRIVATE_CHANNEL_ID> --force` | "Archived channel" |

---

## Part 6: Config Command Tests ⚠️

**Warning:** These tests manipulate your stored tokens. Run last and be prepared to re-authenticate.

### 6.1 Config Show (Dual Token)

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck config show` | Shows both Bot Token and User Token (masked) with storage location |

### 6.2 Config Test (Dual Token)

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck config test` | Tests both bot and user tokens, shows workspace info for each |

### 6.3 Token Type Detection

| Step | Command | Expected |
|------|---------|----------|
| 1 | `printf '%s' xoxb-test-fake-token \| slck set-credential --key bot_token --stdin` | "Stored bot_token" |
| 2 | `printf '%s' xoxp-test-fake-token \| slck set-credential --key user_token --stdin` | "Stored user_token" |
| 3 | `slck config set-token xoxb-x` | Error: removed; points to `slck set-credential` (nonzero exit) |

### 6.4 Selective Token Deletion

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck config delete-token --type bot --force` | "Bot token deleted" |
| 2 | `slck config show` | Bot Token: Not configured, User Token: still present |
| 3 | `slck workspace info` | Error: no bot token configured |
| 4 | `slck search messages "test"` | Still works (uses user token) |
| 5 | `slck config delete-token --type user --force` | "User token deleted" |
| 6 | `slck search messages "test"` | Error: user token required |

### 6.5 Restore Tokens

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck init` | Prompts for bot (and optional user) token, stores them |
| 2 | `printf '%s' "$XOXP" \| slck set-credential --key user_token --stdin` | Stores user token |
| 3 | `slck config show` | Both tokens present (values never shown) |
| 4 | `slck config test` | Both tokens valid |

### 6.6 Delete All Tokens

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck config delete-token --type all --force` | Both tokens deleted |
| 2 | `slck config show` | No tokens configured |

---

## Part 7: Error Handling & Edge Cases

These can be run at any time to verify error handling.

### 7.1 Invalid Input Errors

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck channels get INVALID` | Error with helpful hint |
| 2 | `slck users get INVALID` | Error with helpful hint |
| 3 | `slck messages react $TEST_CHANNEL_ID badts emoji` | Validation error |
| 4 | `slck messages send` | Usage help shown |
| 5 | `slck channels unknown` | Unknown command error |

### 7.2 Permission Errors

| Step | Command | Expected |
|------|---------|----------|
| 1 | `printf '%s' invalid \| slck set-credential --key bot_token --stdin` then `slck workspace info` | Error: `invalid_auth` (env vars are not read at runtime) |
| 2 | (Use token without `channels:manage`) `slck channels create test` | Error describing missing scope |

### 7.3 Edge Cases

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck messages send $TEST_CHANNEL_ID "Hello 👋 世界"` | Unicode preserved |
| 2 | `slck messages send $TEST_CHANNEL_ID 'Test <>&"'"'"' chars'` | Special chars escaped |
| 3 | `slck channels create my-test-with-hyphens` | Works (clean up after) |

---

## Part 8: Emoji & Files Tests

**Scopes required:** `emoji:read`, `files:read`

These are read-only tests with no side effects.

### 8.1 Emoji List

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck emoji list` | Lists custom emoji names (or "No custom emoji found") |
| 2 | `slck emoji list --include-aliases` | Includes alias entries (prefixed with `alias:` in JSON) |
| 3 | `slck emoji list -o json` | Valid JSON map of emoji name → URL |

### 8.2 Files Download

> Requires a known file ID from your workspace. Upload a test file to Slack first, then note its file ID from the message JSON.

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck files download <FILE_ID>` | "Downloaded <name> (<size> bytes) to <path>" |
| 2 | `slck files download <FILE_ID> --output /tmp/test-download` | File saved to specified path |
| 3 | `slck files download <FILE_ID> -o json` | JSON with file_id, name, size, path fields |

---

## Part 9: Identity Tests

No additional scopes required (uses `auth.test` which works with any valid token).

### 9.1 Whoami

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck whoami` | Shows Bot name/ID, Workspace name |
| 2 | `slck whoami -o json` | JSON with bot, workspace fields |

---

## Part 10: Canvas Tests

**Scopes required:** Canvas API scopes (see [Slack Canvas API docs](https://docs.slack.dev/surfaces/canvases/))

These tests create, edit, and delete canvases. Cleanup is included.

### 10.1 Create Standalone Canvas

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `slck canvas create --title "Test Canvas" --text "# Hello\n\nIntegration test"` | "Created canvas: F..." | **Save CANVAS_ID₁** |
| 2 | `slck canvas create --title "Test Canvas" --text "# Hello" -o json` | JSON with `canvas_id` and `title` | (verify only) |

### 10.2 Create Canvas from File

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `echo "# From stdin" \| slck canvas create --title "Stdin Canvas" --file -` | "Created canvas: F..." | **Save CANVAS_ID₂** |

### 10.3 Create Channel Canvas

| Step | Command | Expected | Capture |
|------|---------|----------|---------|
| 1 | `slck canvas create --channel $TEST_CHANNEL_ID --text "# Channel Doc"` | "Created channel canvas: F..." | **Save CANVAS_ID₃** |

### 10.4 Edit Canvas

Using **CANVAS_ID₁** from step 10.1:

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck canvas edit <CANVAS_ID₁> --text "# Updated\n\nNew content"` | "Updated canvas: F..." |
| 2 | `slck canvas edit <CANVAS_ID₁> --text "# Updated again" -o json` | JSON with `canvas_id` and `status: "updated"` |

### 10.5 Error Cases

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck canvas create --text "no title"` | Error: `--title is required for standalone canvases` |
| 2 | `slck canvas create --title "both" --channel C123 --text "x"` | Error: `--title is not used with --channel` |
| 3 | `slck canvas create --title "no content"` | Error: `content required` |
| 4 | `slck canvas edit F12345 ` | Error: `content required` |

### 10.6 Cleanup: Delete Canvases

| Step | Command | Expected |
|------|---------|----------|
| 1 | `slck canvas delete <CANVAS_ID₁>` | "Deleted canvas: F..." |
| 2 | `slck canvas delete <CANVAS_ID₂>` | "Deleted canvas: F..." |
| 3 | `slck canvas delete <CANVAS_ID₃>` | "Deleted canvas: F..." |

---

## Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid_auth` | Token invalid or expired | Regenerate token, run `slck init` (or `slck set-credential`) |
| `not_in_channel` | Bot not in channel | `/invite @botname` in Slack |
| `channel_not_found` | Wrong ID or no access | Verify ID, check bot permissions |
| `missing_scope` | Token lacks required scope | Add scope in Slack app settings, reinstall |
| `ratelimited` | Too many requests | Wait and retry |
| `already_archived` | Channel already archived | Use `channels unarchive` first |
| `name_taken` | Channel name exists | Choose different name |

---

## Quick Reference

```bash
# Show all commands
slck --help

# Command-specific help
slck channels --help
slck messages send --help
slck search --help
slck search messages --help

# Check token configuration
slck config show
slck config test
```

---

## Cleanup Best Practices

**Always clean up test artifacts.** Test messages left in shared channels create noise for other team members.

### Guidelines

1. **Track timestamps**: Save every TS value from message sends (TS₁, TS₂, TS₃, etc.)
2. **Delete immediately after testing**: Don't batch cleanups - delete as you complete each test section
3. **Verify deletion**: Check message history after cleanup to confirm removal
4. **If interrupted**: Note any uncleaned timestamps and delete them before your next session

### Quick Cleanup Commands

```bash
# Delete a single message
slck messages delete $TEST_CHANNEL_ID <timestamp> --force

# Verify it's gone
slck messages history $TEST_CHANNEL_ID --limit 5
```

### What to Clean Up

| Test Section | Artifacts | Cleanup Command |
|--------------|-----------|-----------------|
| Part 3 (Messaging) | Test messages TS₁, TS₂ | `messages delete` |
| Part 3B (Search) | Search test message TS₃ | `messages delete` |
| Part 5 (Destructive) | Test channels | `channels archive` |
| Part 10 (Canvas) | Test canvases CANVAS_ID₁–₃ | `canvas delete` |
