---
allowed-tools: Bash(git:*), Bash(gh:*)
description: Generate release notes from commits and update PR description
---

# Release Notes Generator

Generate comprehensive release notes from commits since the last release.

## Steps

1. Get current branch:
   ```bash
   git branch --show-current
   ```

2. Get commits since main:
   ```bash
   git log origin/main..HEAD --oneline --no-merges
   ```

3. Get diff summary:
   ```bash
   git diff origin/main --stat
   ```

4. Check if PR exists:
   ```bash
   gh pr view --json number,title,body
   ```

## Release Notes Format

Generate notes in this format:

### Summary
One-sentence description of the changes.

### What's New
- Feature descriptions (from feat: commits)
- List each feature with a brief description

### Bug Fixes
- Bug fix descriptions (from fix: commits)
- Reference the issue if applicable

### Breaking Changes
- List any breaking changes (from feat!: or BREAKING CHANGE:)
- Include migration instructions

### Other Changes
- Refactoring, documentation, test improvements

## After Generating

1. Show the generated release notes to the user
2. Ask: "Update PR description with these release notes?"
3. If confirmed, update the PR:
   ```bash
   gh pr edit --body "[generated notes]"
   ```
