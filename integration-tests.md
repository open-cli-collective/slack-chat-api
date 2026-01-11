# Integration Tests

Manual tests for verifying real-world behavior against a live Slack workspace.

## Test Environment Setup

### Prerequisites

- Slack workspace with bot installed
- Bot token (`xoxb-*`) with the following scopes:
  - `channels:read`, `channels:write`, `channels:history`
  - `groups:read`, `groups:write`, `groups:history`
  - `chat:write`, `chat:write.public`
  - `reactions:read`, `reactions:write`
  - `users:read`
  - `team:read`
- Permission to create/archive channels in the workspace
- A test channel (recommend `#slack-cli-test`)

### Environment Setup

```bash
# Option 1: Set token via environment variable
export SLACK_API_TOKEN=xoxb-your-token-here

# Option 2: Store token securely
slack-cli config set-token xoxb-your-token-here

# Verify configuration
slack-cli config show
```

### Test Data Conventions

- Test channels should use `[Test]` prefix in topic/purpose
- Clean up test data (messages, channels) after testing
- Use dedicated test channel to avoid noise in production channels

---

## Command Tests

### Configuration Commands

#### config set-token
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Set valid token | `slack-cli config set-token xoxb-...` | "API token stored..." message |
| Set empty token | `slack-cli config set-token ""` | Error: "token cannot be empty" |
| Interactive input | `slack-cli config set-token` (then enter token) | Prompts for input, stores token |

#### config show
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Token configured | `slack-cli config show` | Shows masked token |
| No token | (clear token first) `slack-cli config show` | "Not configured" message |

#### config delete-token
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete existing | `slack-cli config delete-token` | "API token deleted" message |
| Delete when none | `slack-cli config delete-token` | Error message |

---

### Channel Commands

#### channels list
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all | `slack-cli channels list` | Table with ID, Name, Members |
| JSON output | `slack-cli channels list --json` | Valid JSON array |
| Filter public | `slack-cli channels list --types=public_channel` | Only public channels |
| Filter private | `slack-cli channels list --types=private_channel` | Only private channels (bot must be member) |
| Include archived | `slack-cli channels list --exclude-archived=false` | Includes archived channels |
| Limit results | `slack-cli channels list --limit=5` | Maximum 5 channels |

#### channels get
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get by ID | `slack-cli channels get C01234ABCDE` | Shows channel details |
| JSON output | `slack-cli channels get C01234ABCDE --json` | Valid JSON object |
| Invalid ID | `slack-cli channels get INVALID` | Error: channel_not_found |
| Not a member | `slack-cli channels get C...` (private, not member) | Error: channel_not_found |

#### channels create
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Create public | `slack-cli channels create test-channel` | "Created channel" with ID |
| Create private | `slack-cli channels create test-private --private` | Private channel created |
| Duplicate name | `slack-cli channels create general` | Error: name_taken |
| Invalid name | `slack-cli channels create "has spaces"` | Error: invalid_name_specials |

#### channels archive
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Archive channel | `slack-cli channels archive C...` | "Archived channel" message |
| Already archived | `slack-cli channels archive C...` | Error: already_archived |
| Invalid channel | `slack-cli channels archive INVALID` | Error: channel_not_found |

#### channels unarchive
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Unarchive | `slack-cli channels unarchive C...` | "Unarchived channel" message |
| Not archived | `slack-cli channels unarchive C...` | Error: not_archived |

#### channels set-topic
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Set topic | `slack-cli channels set-topic C... "New topic"` | "Set topic" message |
| Clear topic | `slack-cli channels set-topic C... ""` | Clears topic |
| Not in channel | `slack-cli channels set-topic C...` (not member) | Error: not_in_channel |

#### channels set-purpose
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Set purpose | `slack-cli channels set-purpose C... "Purpose"` | "Set purpose" message |

#### channels invite
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Invite user | `slack-cli channels invite C... U...` | "Invited 1 user(s)" |
| Invite multiple | `slack-cli channels invite C... U1 U2 U3` | "Invited 3 user(s)" |
| Already member | `slack-cli channels invite C... U...` | Error: already_in_channel |

---

### User Commands

#### users list
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| List all | `slack-cli users list` | Table with ID, Name, Real Name |
| JSON output | `slack-cli users list --json` | Valid JSON array |
| Limit results | `slack-cli users list --limit=10` | Maximum 10 users |

#### users get
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get by ID | `slack-cli users get U01234ABCDE` | Shows user details |
| JSON output | `slack-cli users get U01234ABCDE --json` | Valid JSON object |
| Invalid ID | `slack-cli users get INVALID` | Error: user_not_found |

---

### Message Commands

#### messages send
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Send simple | `slack-cli messages send C... "Hello"` | "Message sent (ts: ...)" |
| Send with blocks | `slack-cli messages send C... "Text"` | Block Kit formatted message |
| Send plain | `slack-cli messages send C... "Plain" --simple` | Plain text message |
| Send to thread | `slack-cli messages send C... "Reply" --thread=1234.5678` | Thread reply |
| Invalid channel | `slack-cli messages send INVALID "Text"` | Error: channel_not_found |
| Not in channel | `slack-cli messages send C... "Text"` (private, not member) | Error: not_in_channel |

#### messages update
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Update message | `slack-cli messages update C... 1234.5678 "Updated"` | "Message updated" |
| Invalid ts | `slack-cli messages update C... INVALID "Text"` | Error: message_not_found |
| Not own message | `slack-cli messages update C... 1234.5678 "Text"` | Error: cant_update_message |

#### messages delete
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Delete own | `slack-cli messages delete C... 1234.5678` | "Message deleted" |
| Invalid ts | `slack-cli messages delete C... INVALID` | Error: message_not_found |

#### messages history
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get history | `slack-cli messages history C...` | Table with timestamp, user, text |
| JSON output | `slack-cli messages history C... --json` | Valid JSON array |
| Limit results | `slack-cli messages history C... --limit=5` | Maximum 5 messages |
| Time range | `slack-cli messages history C... --oldest=1234 --latest=5678` | Messages in range |

#### messages thread
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get replies | `slack-cli messages thread C... 1234.5678` | Shows thread messages |
| No replies | `slack-cli messages thread C... 1234.5678` | Empty or parent only |
| Invalid ts | `slack-cli messages thread C... INVALID` | Error: thread_not_found |

#### messages react
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Add reaction | `slack-cli messages react C... 1234.5678 thumbsup` | "Added :thumbsup:" |
| With colons | `slack-cli messages react C... 1234.5678 :heart:` | "Added :heart:" |
| Invalid emoji | `slack-cli messages react C... 1234.5678 notanemoji` | Error: invalid_name |
| Already reacted | `slack-cli messages react C... 1234.5678 thumbsup` | Error: already_reacted |

#### messages unreact
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Remove reaction | `slack-cli messages unreact C... 1234.5678 thumbsup` | "Removed :thumbsup:" |
| No reaction | `slack-cli messages unreact C... 1234.5678 heart` | Error: no_reaction |

---

### Workspace Commands

#### workspace info
| Test Case | Command | Expected Result |
|-----------|---------|-----------------|
| Get info | `slack-cli workspace info` | Shows ID, Name, Domain |
| JSON output | `slack-cli workspace info --json` | Valid JSON object |

---

## Edge Cases

| Test Case | Expected Result |
|-----------|-----------------|
| Unicode channel names | Handled correctly |
| Unicode in messages | Sent/received correctly |
| Very long messages | Sent without truncation (or proper error) |
| Empty results | Graceful "No X found" message |
| Rate limiting | Retry or clear error message |
| Network timeout | Clear error message |
| Invalid token | Error: invalid_auth |
| Expired token | Error: token_revoked or invalid_auth |
| Missing scopes | Error describing missing permission |

---

## Test Execution Checklist

Before each release, verify:

### Setup
- [ ] Build latest version: `make build`
- [ ] Verify token is configured: `./bin/slack-cli config show`
- [ ] Identify test channel ID

### Core Functionality
- [ ] `channels list` returns results
- [ ] `channels get <ID>` shows details
- [ ] `users list` returns results
- [ ] `messages send` delivers message
- [ ] `messages history` shows messages
- [ ] `workspace info` shows workspace

### JSON Output
- [ ] All list commands output valid JSON with `--json`
- [ ] All get commands output valid JSON with `--json`

### Error Handling
- [ ] Invalid channel ID returns helpful error
- [ ] Invalid user ID returns helpful error
- [ ] Permission errors are clear

### Cleanup
- [ ] Delete test messages
- [ ] Archive/delete test channels
- [ ] Document any issues found

---

## Troubleshooting

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid_auth` | Token invalid | Regenerate token, run `config set-token` |
| `not_in_channel` | Bot not in channel | Invite bot with `/invite @botname` |
| `channel_not_found` | Wrong ID or no access | Verify ID, check bot permissions |
| `missing_scope` | Token lacks scope | Reinstall app with required scopes |
| `ratelimited` | Too many requests | Wait and retry |

### Getting Help

```bash
# Show all commands
slack-cli --help

# Show command-specific help
slack-cli channels --help
slack-cli messages send --help
```
