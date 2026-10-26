# slack-chat-api Development Guide

This is the repo-local guide for Slack-specific facts. Shared Open CLI
Collective standards and automation remain canonical in their own repositories.

## Project Overview

slack-chat-api builds the `slck` command-line interface for Slack. It supports
channel management, user lookup, messaging, reactions, message history, and
workspace information.

## Quick Commands

```bash
make build      # build binary to ./bin/slck
make test       # run tests with race detection
make test-cover # run tests with coverage output
make lint       # run golangci-lint
make clean      # remove build artifacts
make install    # install to /usr/local/bin
```

## Repo Structure

```text
slack-chat-api/
├── cmd/slck/main.go
├── internal/
│   ├── cmd/        # Cobra command implementations
│   ├── client/     # Slack API client wrapper
│   ├── keychain/   # cli-common credstore adapter
│   ├── output/     # text/table rendering helpers
│   ├── render/     # message lines shared by messages and watch
│   ├── watch/      # watch store, event application, Socket Mode listener
│   └── version/    # build-time version injection
├── ARCHITECTURE.md # project-level JSON-vs-text contract
├── .golangci.yml
└── Makefile
```

## Command Patterns

- Commands use small options structs with injectable clients for testability.
- New commands live under the appropriate `internal/cmd/<resource>/` package.
- Register new commands from the resource root command.
- Add tests around the command runner with a mock or injectable client.

## Output Contract

Resource and mutation-success commands emit text or table output. JSON is
reserved for local control-plane carve-outs, currently `slck config show --json`,
and for `slck apps get`, whose output is the app's manifest itself (the
document `apps validate`, `diff`, `create` and `update` read back), not a
rendering of it.
The project-level JSON-vs-text contract lives in `ARCHITECTURE.md`.

Source of truth: https://github.com/open-cli-collective/slack-chat-api/blob/main/ARCHITECTURE.md
Local convenience copy, if present: `ARCHITECTURE.md`

Shared output policy:

```md
Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/output-and-rendering.md
Local convenience copy, if present: `../cli-common/docs/output-and-rendering.md`
```

## Credentials And Config

`app_token` holds an app-level token (xapp-) with `connections:write`; only
`slck watch listen` reads it, to open the Socket Mode connection.

`slck` stores credentials in the OS keyring through `cli-common/credstore`.
Non-secret config such as `credential_ref`, `workspace`, and `keyring.backend`
lives in `~/.config/slack-chat-api/config.yml`. Secret ingress is through
`slck init` or `slck set-credential`, using stdin or `--from-env` style inputs.

Shared credential and state policy:

```md
Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-secrets.md
Local convenience copy, if present: `../cli-common/docs/working-with-secrets.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/working-with-state.md
Local convenience copy, if present: `../cli-common/docs/working-with-state.md`
```

## Shared Repo Standards

Use these sources for shared repository policy. Do not copy their mechanics into
this guide.

```md
Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/command-surface.md
Local convenience copy, if present: `../cli-common/docs/command-surface.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/repo-layout.md
Local convenience copy, if present: `../cli-common/docs/repo-layout.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/ci.md
Local convenience copy, if present: `../cli-common/docs/ci.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/release.md
Local convenience copy, if present: `../cli-common/docs/release.md`

Source of truth: https://github.com/open-cli-collective/cli-common/blob/main/docs/distribution.md
Local convenience copy, if present: `../cli-common/docs/distribution.md`
```

## Shared Automation

Use `open-cli-collective/.github` for shared action and reusable workflow
implementations.

```md
Source of truth: https://github.com/open-cli-collective/.github
Local convenience copy, if present: `../.github`
```

## Watch

`slck watch` keeps one file per watched message, and the listener's
`listener.json`, under `watches/` in slck's data directory. Only the listener
(through `watch.Syncer`) changes a watch's messages, so a file's `activity`
always matches them; writers across processes serialize on a `.lock` file in
that directory. `status` and `wait` read the files and never call Slack. They
trust them only while the listener confirmed its connection within
`LiveWithin` (75s; it pings every `PingEvery`, 20s) and fail otherwise. After
a gap longer than `CatchUpAfter` the listener rereads every watched thread
off the read loop and applies again the events that arrived meanwhile.

## Dependencies

- `github.com/open-cli-collective/cli-common` - shared credential storage.
- `github.com/spf13/cobra` - command framework.
- `golang.org/x/term` - no-echo prompt support.
- `github.com/coder/websocket` - the Socket Mode connection.

The Slack HTTP client is implemented in `internal/client`; this repo does not
use `slack-go` or `zalando`.
