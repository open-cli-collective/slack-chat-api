# Changelog

## [3.0.1](https://github.com/piekstra/slack-chat-api/compare/v3.0.0...v3.0.1) (2026-01-12)


### Bug Fixes

* Add ldflags to release builds and auto-merge release PRs ([#39](https://github.com/piekstra/slack-chat-api/issues/39)) ([352a3f1](https://github.com/piekstra/slack-chat-api/commit/352a3f1c42e889f0a1957948fa9c44d6c414b3d0))
* Extract PR number from JSON output for auto-merge ([#41](https://github.com/piekstra/slack-chat-api/issues/41)) ([95fce10](https://github.com/piekstra/slack-chat-api/commit/95fce10aed7c8618040e51409a66b48a3bfc2281))

## [3.0.0](https://github.com/piekstra/slack-chat-api/compare/v2.1.0...v3.0.0) (2026-01-12)


### ⚠ BREAKING CHANGES

* Renamed project from `slack-cli` to `slack-chat-api`
* Binary renamed from `slack-cli` to `slack-chat-api`
* Config directory changed from `~/.config/slack-cli` to `~/.config/slack-chat-api`
* Keychain service name changed


### Features

* Rename project from slack-cli to slack-chat-api ([#37](https://github.com/piekstra/slack-chat-api/issues/37)) ([b03b09e](https://github.com/piekstra/slack-chat-api/commit/b03b09e794f0a41c6f2b1dc1ceed316cebde3b50))

## [2.1.0](https://github.com/piekstra/slack-cli/compare/v2.0.0...v2.1.0) (2026-01-12)


### Features

* Add Slack search functionality with dual token support ([#31](https://github.com/piekstra/slack-cli/issues/31)) ([00c28f9](https://github.com/piekstra/slack-cli/commit/00c28f9f50127b072cec2dfec8044ff31d05dc8f))

## [2.0.0](https://github.com/piekstra/slack-cli/compare/v1.2.0...v2.0.0) (2026-01-12)


### ⚠ BREAKING CHANGES

* Replace --json flag with --output flag

### Features

* Add --output flag and consistent output formatting (Phase 4) ([#15](https://github.com/piekstra/slack-cli/issues/15)) ([574008b](https://github.com/piekstra/slack-cli/commit/574008b5ee5a87abd39af0bd6d8108474421b151))
* Add Claude Code AI agent configuration ([#18](https://github.com/piekstra/slack-cli/issues/18)) ([eb05acb](https://github.com/piekstra/slack-cli/commit/eb05acb9052d89455252bf37e042e2a8c139f3f3)), closes [#7](https://github.com/piekstra/slack-cli/issues/7)
* Add config test command and document 1Password integration ([#22](https://github.com/piekstra/slack-cli/issues/22)) ([#23](https://github.com/piekstra/slack-cli/issues/23)) ([72fd833](https://github.com/piekstra/slack-cli/commit/72fd833bd44cfbb9b86092900ac082d814aac705))
* Add input validation, stdin support, and --force flag ([#19](https://github.com/piekstra/slack-cli/issues/19)) ([413865a](https://github.com/piekstra/slack-cli/commit/413865aa2612be0b5138bb3b891e190d59672767)), closes [#8](https://github.com/piekstra/slack-cli/issues/8)
* Add internal/version package for build-time version info ([cf5613e](https://github.com/piekstra/slack-cli/commit/cf5613ec682bd16ae79aaba6c76e3d5e9a46008b))
* Update root command to use version package ([90b6d95](https://github.com/piekstra/slack-cli/commit/90b6d9546eb0e15c21213ed7b5df448b8f01661e))


### Bug Fixes

* Add helpful error for unarchive limitation ([#26](https://github.com/piekstra/slack-cli/issues/26)) ([#29](https://github.com/piekstra/slack-cli/issues/29)) ([e904f11](https://github.com/piekstra/slack-cli/commit/e904f112037a91074aa06b5276d6e2ce4ae885ca))
* Handle token source correctly in config commands ([#25](https://github.com/piekstra/slack-cli/issues/25)) ([#28](https://github.com/piekstra/slack-cli/issues/28)) ([90fb754](https://github.com/piekstra/slack-cli/commit/90fb75456c1955d5c3c68d6335c6c4db480e6014))
* Respect --limit flag in channels/users list ([#24](https://github.com/piekstra/slack-cli/issues/24)) ([#27](https://github.com/piekstra/slack-cli/issues/27)) ([2168989](https://github.com/piekstra/slack-cli/commit/21689893d65bfd10ec8488e0dc3432cfae70da71))
